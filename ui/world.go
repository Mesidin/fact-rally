package ui

import (
	"embed"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/bike_ride.png assets/bike_lean.png assets/bike_wheelie.png assets/bike_park.png
var bikeFiles embed.FS

//go:embed assets/prop_tree.png assets/prop_lamp.png assets/prop_board.png assets/prop_plus.png assets/prop_post.png assets/prop_bar.png
var propFiles embed.FS

var skyStars = func() []star {
	r := rand.New(rand.NewPCG(7, 11))
	s := make([]star, 42)
	span := horizon * 58 / 100
	for i := range s {
		s[i] = star{
			x: float32(r.IntN(ScreenW)),
			y: float32(6 + r.IntN(span)),
			r: 1 + float32(r.IntN(2)),
			f: 1.2 + float64(r.IntN(16))/5,
		}
	}
	return s
}()

// skyline is the night city. The middle is left open for the sun.
var skyline = []tower{
	{x: 8, w: 54, h: 92},
	{x: 68, w: 36, h: 150},
	{x: 110, w: 78, h: 118},
	{x: 196, w: 28, h: 128},
	{x: 230, w: 64, h: 108},
	{x: 302, w: 48, h: 96},
	{x: 360, w: 40, h: 70},
	{x: 860, w: 44, h: 78},
	{x: 912, w: 70, h: 140},
	{x: 990, w: 30, h: 132},
	{x: 1026, w: 86, h: 112},
	{x: 1120, w: 40, h: 124},
	{x: 1168, w: 96, h: 108},
}

type star struct {
	x, y, r float32
	f       float64
}

type tower struct {
	x, w, h float32
}

// ridePose is the motorcycle for one frame.
type ridePose struct {
	show    bool
	lane    float64 // 0 is the middle. Positive is toward the right shoulder.
	wheelie float64
	lean    float64
	sparks  float64
	park    bool
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

func withA(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

func smooth(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

// curveAt bends the road ahead of the rider. t is 0 at the horizon and 1
// under the wheels, so the lane at the motorcycle stays put while the
// distance swings left and right.
func curveAt(t, phase float64) float64 {
	d := 1 - t
	if d < 0 {
		d = 0
	}
	d *= d
	ang := phase*1.35 + 0.85
	bend := math.Sin(ang) + 0.4*math.Sin(ang*0.45+1.9)
	return bend * 270 * d
}

// bendLean banks the rider into the curve ahead. The horizon sits left of
// the bike on a left-hand bend, and that lean is negative.
func bendLean(phase float64) float64 {
	return curveAt(0.42, phase) / 240
}

// roadX is a point on the road edge. t is 0 at the horizon and 1 at the bottom.
// side is -1 for the left edge and +1 for the right.
func roadX(t, side, shake, phase float64) (x, y float32) {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	y = horizon + float32(t)*float32(ScreenH-horizon)
	half := 18 + (500-18)*t
	x = float32(float64(ScreenW)/2 + shake + curveAt(t, phase) + side*half)
	return x, y
}

func drawWorld(dst *ebiten.Image, phase, near, boost, burst, clock, shake float64, gateNum int, gateT float64, showGate bool, pose ridePose) {
	drawSky(dst, clock, shake)
	drawRoad(dst, phase, shake, near, pose)
	drawRoadside(dst, phase, shake, gateNum, gateT, showGate)
	hurry := 0.0
	if near > 0.72 {
		hurry = (near - 0.72) / 0.28
	}
	if showGate && gateT > 0.02 {
		drawGate(dst, gateT, hurry, clock, gateNum, shake, phase)
	}
	if burst > 0 {
		bt := gateT
		if bt < 0.02 {
			bt = 0.66
		}
		gx, gy := roadX(bt, 0, shake, phase)
		r := float32((1-burst)*240 + 16)
		vector.StrokeCircle(dst, gx, gy-30, r, 4, withA(color.RGBA{255, 43, 214, 255}, uint8(burst*210)), true)
	}
	if pose.show {
		drawBike(dst, pose, shake, boost, phase)
	}
}

func drawSky(dst *ebiten.Image, clock, shake float64) {
	top := color.RGBA{8, 4, 24, 255}
	mid := color.RGBA{48, 10, 72, 255}
	hot := color.RGBA{186, 24, 96, 255}
	const bands = 28
	for i := 0; i < bands; i++ {
		y0 := horizon * i / bands
		y1 := horizon * (i + 1) / bands
		u := float64(i) / float64(bands-1)
		col := lerp(top, mid, u)
		if u > 0.45 {
			col = lerp(mid, hot, (u-0.45)/0.55)
		}
		vector.FillRect(dst, 0, float32(y0), ScreenW, float32(y1-y0+1), col, false)
	}
	drawSun(dst, ScreenW/2+float32(shake*0.15), horizon+6, 132)
	drawCity(dst, clock, shake)
	cx := float32(ScreenW/2) + float32(shake*0.15)
	for _, s := range skyStars {
		dx := s.x - cx
		dy := s.y - float32(horizon+6)
		if dx*dx+dy*dy < 150*150 {
			continue
		}
		tw := 0.45 + 0.55*(0.5+0.5*math.Sin(clock*s.f))
		vector.FillCircle(dst, s.x+float32(shake*0.2), s.y, s.r, withA(color.RGBA{236, 230, 255, 255}, uint8(70+150*tw)), true)
	}
	vector.FillRect(dst, 0, horizon-10, ScreenW, 16, withA(color.RGBA{255, 43, 214, 255}, 40), false)
	vector.FillRect(dst, 0, horizon-1, ScreenW, 2, color.RGBA{255, 120, 214, 220}, false)
}

func drawSun(dst *ebiten.Image, cx, cy, r float32) {
	hot := color.RGBA{255, 236, 190, 255}
	edge := color.RGBA{255, 70, 128, 255}
	const step = 13
	for y := cy - r; y < cy+r; y += step {
		dy := y + 4 - cy
		if dy < -r || dy > r {
			continue
		}
		half := float32(math.Sqrt(float64(r*r - dy*dy)))
		u := math.Abs(float64(dy)) / float64(r)
		vector.FillRect(dst, cx-half, y, half*2, 7, lerp(hot, edge, u*0.85), false)
	}
}

func drawCity(dst *ebiten.Image, clock, shake float64) {
	body := color.RGBA{10, 6, 22, 255}
	sx := float32(shake * 0.12)
	for i, b := range skyline {
		x := b.x + sx
		y := float32(horizon) - b.h
		vector.FillRect(dst, x, y, b.w, b.h, body, true)
		lit := []color.RGBA{
			{120, 246, 255, 230},
			{255, 74, 216, 230},
			{255, 214, 140, 220},
		}
		for wy := y + 8; wy < float32(horizon)-8; wy += 9 {
			for wx := x + 6; wx < x+b.w-6; wx += 8 {
				n := int(wx)*3 + int(wy)*5 + i*13
				if n%5 == 0 {
					continue
				}
				col := lit[n%3]
				blink := 0.5 + 0.5*math.Sin(clock*(1.5+float64(n%5))+float64(n))
				if n%11 == 0 && blink > 0.82 {
					continue
				}
				vector.FillRect(dst, wx, wy, 3, 4, col, false)
			}
		}
		if b.h > 160 {
			vector.StrokeLine(dst, x+b.w/2, y, x+b.w/2, y-28, 2, color.RGBA{180, 250, 255, 255}, true)
			vector.FillCircle(dst, x+b.w/2, y-30, 3, color.RGBA{255, 60, 140, 255}, true)
		}
	}
}

func drawRoad(dst *ebiten.Image, phase, shake, near float64, pose ridePose) {
	vector.FillRect(dst, 0, horizon, ScreenW, ScreenH-horizon, color.RGBA{18, 6, 32, 255}, false)
	hurry := 0.0
	if near > 0.72 {
		hurry = (near - 0.72) / 0.28
	}
	left := color.RGBA{61, 240, 255, 255}
	right := color.RGBA{255, 43, 214, 255}
	if hurry > 0 {
		hot := color.RGBA{255, 59, 78, 255}
		left = lerp(left, hot, hurry)
		right = lerp(right, hot, hurry)
	}
	if pose.sparks > 0.2 {
		hot := color.RGBA{255, 244, 220, 255}
		if pose.lane >= 0 {
			right = lerp(right, hot, pose.sparks)
		} else {
			left = lerp(left, hot, pose.sparks)
		}
	}
	for y := horizon; y < ScreenH; y += 3 {
		t := float64(y-horizon) / float64(ScreenH-horizon)
		xl, _ := roadX(t, -1, shake, phase)
		xr, _ := roadX(t, 1, shake, phase)
		shoulder := float32(14 + 64*t)
		vector.FillRect(dst, xl-shoulder, float32(y), xr-xl+shoulder*2, 3, color.RGBA{28, 8, 42, 255}, false)
		road := lerp(color.RGBA{36, 28, 64, 255}, color.RGBA{12, 8, 22, 255}, t*0.85)
		vector.FillRect(dst, xl, float32(y), xr-xl, 3, road, false)
		ew := float32(2 + 5*t)
		glow := float32(5 + 14*t)
		vector.FillRect(dst, xl-glow, float32(y), glow, 3, withA(left, 70), false)
		vector.FillRect(dst, xr, float32(y), glow, 3, withA(right, 70), false)
		vector.FillRect(dst, xl, float32(y), ew, 3, left, false)
		vector.FillRect(dst, xr-ew, float32(y), ew, 3, right, false)
		u := t*8 - phase
		frac := u - math.Floor(u)
		if frac < 0.38 && t > 0.05 {
			dw := float32(3 + 14*t)
			cx := (xl + xr) / 2
			vector.FillRect(dst, cx-dw/2, float32(y), dw, 3, lerp(color.RGBA{255, 210, 240, 255}, right, 0.35), false)
		}
	}
}

// gateDepth places the gate from how many problems in this leg are already
// solved. The first problem hides it. The second shows it small. Each
// answer after that jumps it closer. t is 0 at the horizon and 1 at the wheels.
func gateDepth(solved int) (show bool, t float64) {
	switch solved {
	case 0:
		return false, 0
	case 1:
		return true, 0.08
	case 2:
		return true, 0.26
	case 3:
		return true, 0.46
	default:
		if solved < 0 {
			return false, 0
		}
		return true, 0.68
	}
}

const (
	propTree = iota
	propLamp
	propBoard
	propPlus
)

func sceneryGate(gate int) int {
	if gate < 1 {
		return 1
	}
	return gate
}

func sloganFor(gate int) string {
	lines := []string{
		"Math rocks!",
		"You got this!",
		"Keep going!",
		"Facts fly!",
		"Nice pace!",
		"Bank it!",
	}
	g := sceneryGate(gate)
	return lines[(g-1)%len(lines)]
}

// propKindFor picks the object in a roadside slot. Slot 3 is that gate's
// billboard. The other slots change with the gate: trees, then lamps,
// then plus signs, then a mix.
func propKindFor(gate, slot int) int {
	if slot == 3 {
		return propBoard
	}
	switch (sceneryGate(gate) - 1) % 4 {
	case 0:
		if slot%4 == 1 {
			return propLamp
		}
		return propTree
	case 1:
		if slot%4 == 0 {
			return propTree
		}
		return propLamp
	case 2:
		if slot%4 == 0 {
			return propTree
		}
		return propPlus
	default:
		if slot%2 == 0 {
			return propLamp
		}
		return propTree
	}
}

type roadsideProp struct {
	t    float64
	side float64
	kind int
}

func drawRoadside(dst *ebiten.Image, phase, shake float64, gate int, gateT float64, showGate bool) {
	const n = 8
	props := make([]roadsideProp, n)
	for i := 0; i < n; i++ {
		t := math.Mod(float64(i)/float64(n)+phase*0.22, 1)
		side := -1.0
		if i%2 == 1 {
			side = 1
		}
		props[i] = roadsideProp{t: t, side: side, kind: propKindFor(gate, i)}
	}
	slices.SortFunc(props, func(a, b roadsideProp) int {
		if a.t < b.t {
			return -1
		}
		if a.t > b.t {
			return 1
		}
		return 0
	})
	block := showGate && gateT > 0.02
	var gateL, gateR float32
	if block {
		gateL, _ = roadX(gateT, -1, shake, phase)
		gateR, _ = roadX(gateT, 1, shake, phase)
		pad := float32(12 + 8*propScale(gateT))
		gateL -= pad
		gateR += pad
	}
	var boards []roadsideProp
	for _, p := range props {
		if p.t < 0.14 || p.t > 0.92 {
			continue
		}
		if p.kind == propBoard {
			boards = append(boards, p)
			continue
		}
		drawOneProp(dst, p, shake, phase)
	}
	// The slogan is drawn after the other props, and outside the gate, so a
	// lamp or the arch does not cover the words.
	for _, p := range boards {
		if block {
			leftRoom := gateL - 12
			rightRoom := float32(ScreenW) - 12 - gateR
			if p.side > 0 && rightRoom < 56 && leftRoom >= 56 {
				p.side = -1
			} else if p.side < 0 && leftRoom < 56 && rightRoom >= 56 {
				p.side = 1
			}
		}
		scale := propScale(p.t)
		lane, y := roadX(p.t, p.side, shake, phase)
		out := float32(20+78*p.t) + 42 + 48*float32(p.t)
		x := lane + float32(p.side)*out
		drawBoard(dst, x, y, scale, sloganFor(gate), p.side, gateL, gateR, block)
	}
}

func drawOneProp(dst *ebiten.Image, p roadsideProp, shake, phase float64) {
	scale := propScale(p.t)
	lane, y := roadX(p.t, p.side, shake, phase)
	out := float32(20 + 78*p.t)
	x := lane + float32(p.side)*out
	switch p.kind {
	case propLamp:
		drawLamp(dst, x, y, scale, p.side > 0)
	case propPlus:
		drawPlus(dst, x, y, scale, p.side > 0)
	default:
		drawTree(dst, x, y, scale, p.side > 0)
	}
}

// Roadside sprites share the night palette. They are drawn with nearest
// filtering at whole-pixel scales, so the chunks stay square.
//
// The palm's anchor is the bottom of the trunk. The lamp's arm points to +x,
// so the left shoulder draws it unflipped and the right shoulder mirrors it.
// The billboard face is a dark rectangle; the slogan is live text on top.
const (
	treeAnchorX  = 17
	treeAnchorY  = 57
	lampAnchorX  = 4
	lampAnchorY  = 63
	boardAnchorX = 26
	boardAnchorY = 39
	boardFaceX   = 3
	boardFaceY   = 3
	boardFaceW   = 46
	boardFaceH   = 16
	plusAnchorX  = 8
	plusAnchorY  = 29
	postAnchorX  = 4
	postAnchorY  = 51
)

var tintNone = color.RGBA{255, 255, 255, 255}

// propScale is the nearest-neighbor step for a roadside object at depth t.
// t is 0 at the horizon and 1 under the wheels.
func propScale(t float64) int {
	s := int(math.Round(1 + t*3))
	if s < 1 {
		return 1
	}
	if s > 4 {
		return 4
	}
	return s
}

// fitBoard parks the sign in the shoulder gap: on screen, and outside the
// gate when the arch is up. It shrinks the scale until the sign fits.
func fitBoard(x float32, side float64, gateL, gateR float32, block bool, srcW, scale int) (float32, int) {
	const margin = 12
	for s := scale; s >= 1; s-- {
		w := float32(srcW * s)
		leftLim := float32(margin)
		rightLim := float32(ScreenW) - margin
		if block {
			if side > 0 {
				leftLim = gateR + 4
			} else {
				rightLim = gateL - 4
			}
		}
		if leftLim < margin {
			leftLim = margin
		}
		if rightLim > float32(ScreenW)-margin {
			rightLim = float32(ScreenW) - margin
		}
		if rightLim-leftLim < w {
			continue
		}
		left := x - float32(boardAnchorX*s)
		if left < leftLim {
			left = leftLim
		}
		if left+w > rightLim {
			left = rightLim - w
		}
		return left + float32(boardAnchorX*s), s
	}
	return x, 1
}

// fitSprite drops the scale until the sprite top stays under the skyline.
func fitSprite(ground float32, srcH, scale int) int {
	if srcH < 1 || scale < 1 {
		return 0
	}
	limit := float32(horizon) + 8
	for scale > 1 && ground-float32(srcH*scale) < limit {
		scale--
	}
	if ground-float32(srcH*scale) < limit {
		return 0
	}
	return scale
}

func drawTree(dst *ebiten.Image, x, ground float32, scale int, flip bool) {
	loadProps()
	scale = fitSprite(ground, sprTree.Bounds().Dy(), scale)
	if scale == 0 {
		return
	}
	blitProp(dst, sprTree, treeAnchorX, treeAnchorY, float64(x), float64(ground), scale, flip, tintNone)
}

func drawLamp(dst *ebiten.Image, x, ground float32, scale int, flip bool) {
	loadProps()
	scale = fitSprite(ground, sprLamp.Bounds().Dy(), scale)
	if scale == 0 {
		return
	}
	blitProp(dst, sprLamp, lampAnchorX, lampAnchorY, float64(x), float64(ground), scale, flip, tintNone)
}

func drawBoard(dst *ebiten.Image, x, ground float32, scale int, slogan string, side float64, gateL, gateR float32, blockGate bool) {
	loadProps()
	scale = fitSprite(ground, sprBoard.Bounds().Dy(), scale)
	if scale == 0 {
		return
	}
	x, scale = fitBoard(x, side, gateL, gateR, blockGate, sprBoard.Bounds().Dx(), scale)
	blitProp(dst, sprBoard, boardAnchorX, boardAnchorY, float64(x), float64(ground), scale, false, tintNone)
	s := float32(scale)
	fx := float32(math.Round(float64(x))) + float32(boardFaceX-boardAnchorX)*s
	fy := float32(math.Round(float64(ground))) + float32(boardFaceY-boardAnchorY)*s
	drawSlogan(dst, slogan, fx, fy, float32(boardFaceW)*s, float32(boardFaceH)*s)
}

func drawSlogan(dst *ebiten.Image, slogan string, x, y, w, h float32) {
	if slogan == "" || w < 12 || h < 8 {
		return
	}
	size := float64(h) * 0.72
	var tw, th float64
	for size >= 8 {
		tw, th = text.Measure(slogan, face(size), 0)
		if float32(tw) <= w-4 && th <= float64(h) {
			break
		}
		size -= 0.5
	}
	if size < 8 || float32(tw) > w-4 {
		return
	}
	top := y + (h-float32(th))/2
	drawLabel(dst, slogan, x+w/2, top, float32(size), colInk, text.AlignCenter)
}

func drawPlus(dst *ebiten.Image, x, ground float32, scale int, warm bool) {
	loadProps()
	scale = fitSprite(ground, sprPlus.Bounds().Dy(), scale)
	if scale == 0 {
		return
	}
	tint := colCyan
	if warm {
		tint = colAccent
	}
	blitProp(dst, sprPlus, plusAnchorX, plusAnchorY, float64(x), float64(ground), scale, false, tint)
}

func drawGate(dst *ebiten.Image, t, hurry, clock float64, num int, shake, phase float64) {
	loadProps()
	xl, y := roadX(t, -1, shake, phase)
	xr, _ := roadX(t, 1, shake, phase)
	if xr-xl < 8 {
		return
	}
	scale := propScale(t)
	postSrcH := sprPost.Bounds().Dy()
	for scale > 1 && y-float32(postSrcH*scale) < 24 {
		scale--
	}
	tint := tintNone
	labelCol := colAccent
	if hurry > 0 {
		pulse := 0.55 + 0.45*math.Sin(clock*9)
		k := hurry * pulse
		tint = lerp(tintNone, color.RGBA{255, 64, 72, 255}, k)
		labelCol = lerp(colAccent, colRed, k)
	}
	gxL := math.Round(float64(xl))
	gxR := math.Round(float64(xr))
	gy := math.Round(float64(y))
	pw := sprPost.Bounds().Dx() * scale
	bw := sprBar.Bounds().Dx() * scale
	bh := sprBar.Bounds().Dy() * scale
	postH := postSrcH * scale
	left := int(gxL) - pw/2
	right := int(gxR) + pw/2
	top := int(gy) - postH
	for x := left; x < right; x += bw {
		blitProp(dst, sprBar, 0, 0, float64(x), float64(top), scale, false, tint)
	}

	bannerTop := float32(top + bh)
	bannerH := float32(postH) * 0.40
	if bannerH < float32(8*scale) {
		bannerH = float32(8 * scale)
	}
	maxH := float32(gy) - bannerTop - float32(4*scale)
	if bannerH > maxH {
		bannerH = maxH
	}
	bx := float32(int(gxL) + pw/2)
	br := float32(int(gxR) - pw/2)
	if bannerH >= 4 && br-bx >= 4 {
		vector.FillRect(dst, bx, bannerTop, br-bx, bannerH, color.RGBA{8, 4, 16, 230}, true)
		ch := float32(3 * scale)
		if ch > bannerH*0.4 {
			ch = bannerH * 0.4
		}
		cw := float32(4 * scale)
		by := bannerTop + bannerH - ch
		dark := false
		for x := bx; x < br-0.5; x += cw {
			c := color.RGBA{226, 250, 255, 255}
			if dark {
				c = color.RGBA{14, 8, 22, 255}
			}
			w := cw
			if x+w > br {
				w = br - x
			}
			vector.FillRect(dst, x, by, w, ch, c, false)
			dark = !dark
		}
		if t > 0.16 {
			slot := bannerH - ch
			size := slot * 0.62
			if size >= 11 {
				drawLabel(dst, fmt.Sprintf("GATE %d", num), (bx+br)/2, bannerTop+(slot-size)*0.5, size, labelCol, text.AlignCenter)
			}
		}
	}
	blitProp(dst, sprPost, postAnchorX, postAnchorY, gxL, gy, scale, false, tint)
	blitProp(dst, sprPost, postAnchorX, postAnchorY, gxR, gy, scale, false, tint)
}

// The four poses share one canvas. The anchor is the bottom of the rear tire,
// so a lean, a flare, and a parked bike stay planted on the same contact.
const (
	bikeAnchorX = 431
	bikeAnchorY = 868
	bikeScale   = 0.20
	bikeReach   = 423 // source pixels from the tire out to the far side of a lean
	leanEnter   = 0.26
)

// leanSprite picks the banking drawing. The source art hangs the rider's
// body to the viewer's left of the rear tire, so that frame is the left lean.
// A right lean mirrors it.
func leanSprite(lean float64) (use bool, flip bool) {
	if lean > leanEnter {
		return true, true
	}
	if lean < -leanEnter {
		return true, false
	}
	return false, false
}

var (
	bikeOnce sync.Once
	bikeRide *ebiten.Image
	bikeLean *ebiten.Image
	bikeUp   *ebiten.Image
	bikeStop *ebiten.Image
)

func loadBikes() {
	bikeOnce.Do(func() {
		bikeRide = mustBike("assets/bike_ride.png")
		bikeLean = mustBike("assets/bike_lean.png")
		bikeUp = mustBike("assets/bike_wheelie.png")
		bikeStop = mustBike("assets/bike_park.png")
	})
}

func mustBike(name string) *ebiten.Image {
	return mustPNG(bikeFiles, name)
}

var (
	propOnce sync.Once
	sprTree  *ebiten.Image
	sprLamp  *ebiten.Image
	sprBoard *ebiten.Image
	sprPlus  *ebiten.Image
	sprPost  *ebiten.Image
	sprBar   *ebiten.Image
)

func loadProps() {
	propOnce.Do(func() {
		sprTree = mustPNG(propFiles, "assets/prop_tree.png")
		sprLamp = mustPNG(propFiles, "assets/prop_lamp.png")
		sprBoard = mustPNG(propFiles, "assets/prop_board.png")
		sprPlus = mustPNG(propFiles, "assets/prop_plus.png")
		sprPost = mustPNG(propFiles, "assets/prop_post.png")
		sprBar = mustPNG(propFiles, "assets/prop_bar.png")
	})
}

func mustPNG(files embed.FS, name string) *ebiten.Image {
	f, err := files.Open(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	return ebiten.NewImageFromImage(img)
}

func blitProp(dst, src *ebiten.Image, anchorX, anchorY int, x, y float64, scale int, flip bool, tint color.RGBA) {
	if src == nil || scale < 1 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	sx := float64(scale)
	if flip {
		sx = -sx
	}
	op.GeoM.Translate(float64(-anchorX), float64(-anchorY))
	op.GeoM.Scale(sx, float64(scale))
	op.GeoM.Translate(math.Round(x), math.Round(y))
	op.ColorScale.ScaleWithColor(tint)
	dst.DrawImage(src, op)
}

func drawBike(dst *ebiten.Image, pose ridePose, shake, _ float64, phase float64) {
	loadBikes()
	base := float32(ScreenH - 16)
	t := float64(base-float32(horizon)) / float64(ScreenH-horizon)
	xl, _ := roadX(t, -1, shake, phase)
	xr, _ := roadX(t, 1, shake, phase)
	mid := (xl + xr) / 2
	half := (xr - xl) / 2
	along := pose.lane
	if along > 1 {
		along = 1
	}
	if along < -1 {
		along = -1
	}
	reach := float32(bikeReach) * bikeScale
	cx := mid + float32(along)*half*0.62
	if pose.lane > 1 {
		u := float32(pose.lane-1) / 0.18
		if u > 1 {
			u = 1
		}
		dest := xr - reach*0.22
		cx += (dest - cx) * u
	}
	if cx < reach+12 {
		cx = reach + 12
	}
	if cx > float32(ScreenW)-reach-12 {
		cx = float32(ScreenW) - reach - 12
	}

	lift := 0.0
	if !pose.park {
		lift = pose.wheelie * 11
	}
	ax := float64(cx)
	ay := float64(base) - lift

	if pose.park {
		blitBike(dst, bikeStop, ax, ay, 1, false)
	} else {
		use, flip := leanSprite(pose.lean)
		body := bikeRide
		if use {
			body = bikeLean
		}
		w := pose.wheelie
		if pose.sparks > 0.2 {
			w *= 1 - pose.sparks
		}
		if w < 0 {
			w = 0
		}
		if w > 1 {
			w = 1
		}
		// Banking and the flare are different drawings. A miss keeps the lean.
		if body != bikeRide {
			w = 0
		}
		blitBike(dst, body, ax, ay, 1-w, flip)
		if w > 0.02 {
			blitBike(dst, bikeUp, ax, ay, w, false)
		}
	}

	if pose.sparks > 0.08 {
		drawSparks(dst, float32(cx), base, pose, reach)
	}
}

func blitBike(dst, src *ebiten.Image, ax, ay, alpha float64, flip bool) {
	if src == nil || alpha < 0.02 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	sx := float64(bikeScale)
	if flip {
		sx = -sx
	}
	op.GeoM.Translate(-bikeAnchorX, -bikeAnchorY)
	op.GeoM.Scale(sx, bikeScale)
	op.GeoM.Translate(ax, ay)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(src, op)
}

func drawSparks(dst *ebiten.Image, cx, base float32, pose ridePose, reach float32) {
	side := float32(1)
	if pose.lane < 0 {
		side = -1
	}
	originX := cx + side*reach*0.55
	if originX < 20 {
		originX = 20
	}
	if originX > ScreenW-20 {
		originX = ScreenW - 20
	}
	originY := base - 96
	vector.FillCircle(dst, originX, originY, 6, color.RGBA{255, 236, 180, 240}, true)
	for i := 0; i < 5; i++ {
		u := float32(i)
		length := (16 + u*8) * float32(pose.sparks)
		endX := originX + side*length
		if endX < 8 {
			endX = 8
		}
		if endX > ScreenW-8 {
			endX = ScreenW - 8
		}
		vector.StrokeLine(dst, originX, originY, endX, originY-6-u*5, 3, color.RGBA{255, 196, 70, 230}, true)
	}
}
