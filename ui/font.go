package ui

import (
	"bytes"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	ScreenW = 1280
	ScreenH = 720
	horizon = 312
)

var (
	colHUD    = color.RGBA{8, 4, 18, 255}
	colBtn    = color.RGBA{16, 8, 32, 255}
	colInk    = color.RGBA{244, 241, 234, 255}
	colMuted  = color.RGBA{168, 156, 196, 255}
	colAccent = color.RGBA{255, 43, 214, 255}
	colCyan   = color.RGBA{61, 240, 255, 255}
	colRed    = color.RGBA{255, 59, 78, 255}
)

var faceSrc *text.GoTextFaceSource

func init() {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	faceSrc = src
}

func face(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: faceSrc, Size: size}
}

func drawLabel(dst *ebiten.Image, s string, x, y, size float32, col color.Color, align text.Align) {
	if s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(col)
	op.PrimaryAlign = align
	text.Draw(dst, s, face(float64(size)), op)
}

type button struct {
	x, y, w, h float32
	label, sub string
	id         string
}

func (b button) contains(x, y int) bool {
	fx, fy := float32(x), float32(y)
	return fx >= b.x && fy >= b.y && fx < b.x+b.w && fy < b.y+b.h
}

func drawNeonTitle(dst *ebiten.Image, s string, x, y, size float32) {
	drawLabel(dst, s, x-3, y, size, withA(colAccent, 210), text.AlignCenter)
	drawLabel(dst, s, x+3, y, size, withA(colCyan, 210), text.AlignCenter)
	drawLabel(dst, s, x, y, size, colInk, text.AlignCenter)
}

func drawBtn(dst *ebiten.Image, b button, hot bool) {
	vector.FillRect(dst, b.x, b.y, b.w, b.h, color.RGBA{10, 6, 22, 255}, true)
	rim := withA(colCyan, 150)
	label := colInk
	if hot {
		rim = colAccent
		vector.StrokeRect(dst, b.x-3, b.y-3, b.w+6, b.h+6, 6, withA(colAccent, 50), true)
		label = colInk
	}
	vector.StrokeRect(dst, b.x+1.5, b.y+1.5, b.w-3, b.h-3, 2, rim, true)
	if b.sub == "" {
		drawLabel(dst, b.label, b.x+b.w/2, b.y+b.h/2-14, 22, label, text.AlignCenter)
		return
	}
	drawLabel(dst, b.label, b.x+b.w/2, b.y+12, 22, label, text.AlignCenter)
	drawLabel(dst, b.sub, b.x+b.w/2, b.y+40, 16, colMuted, text.AlignCenter)
}

func commas(n int) string {
	if n < 0 {
		return "-" + commas(-n)
	}
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
