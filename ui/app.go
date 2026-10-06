package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"mathskill/engine"
	"mathskill/internal/meta"
	"mathskill/profile"
)

type scene int

const (
	sceneName scene = iota
	sceneMenu
	sceneCourse
	sceneBand
	sceneCountdown
	sceneRace
	sceneResult
	sceneScores
	sceneHall
	sceneRacers
)

type floater struct {
	text      string
	x, y      float32
	life, max float64
	size      float32
}

type item struct {
	label, sub, id string
}

const pullDur = 1.15

// App is the window. The night city is the game's own look.
type App struct {
	scene      scene
	store      *profile.Store
	racer      string
	name       []rune
	op         engine.Op
	band       engine.Band
	race       *engine.Race
	cursor     int
	count      float64
	clock      float64
	phase      float64
	boost      float64
	shake      float64
	burst      float64
	gateDist   float64
	wrongUntil float64
	lane       float64
	pullFrom   float64
	pullAge    float64
	sparks     float64
	bump       int
	pulling    bool
	bailed     bool
	newBest    bool
	wantQuit   bool
	mx, my     int
	floaters   []floater
	buttons    []button
	music      *playlist
}

func NewApp(store *profile.Store) *App {
	if store == nil {
		store = profile.Empty()
	}
	store.Normalize()
	a := &App{store: store, racer: store.Active, mx: -1, my: -1}
	if a.racer == "" && len(store.Profiles) > 0 {
		a.racer = store.Profiles[0].ID
	}
	if a.racer == "" {
		a.setScene(sceneName)
	} else {
		a.setScene(sceneMenu)
	}
	a.music = newPlaylist()
	return a
}

func (a *App) Layout(int, int) (int, int) { return ScreenW, ScreenH }

func (a *App) Update() error {
	if a.wantQuit {
		return ebiten.Termination
	}
	dt := frameDT()
	sec := dt.Seconds()
	a.clock += sec
	a.decay(sec)
	a.music.update(a.scene != sceneName)

	chars := ebiten.AppendInputChars(nil)
	clicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	mx, my := ebiten.CursorPosition()
	a.followMouse(mx, my)

	switch a.scene {
	case sceneName:
		a.phase += sec * 0.07
		a.updateName(chars)
		if a.scene == sceneName {
			a.clickButtons(clicked, mx, my)
		}
	case sceneCountdown:
		a.updateCountdown(dt)
		a.approachGate(sec)
	case sceneRace:
		a.updateRace(dt)
		a.approachGate(sec)
	default:
		a.phase += sec * 0.07
		a.updatePick(clicked, mx, my)
	}
	a.syncEngine()
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(colHUD)
	mx, my := ebiten.CursorPosition()

	switch a.scene {
	case sceneCountdown, sceneRace, sceneResult:
		a.drawRally(screen)
	default:
		drawWorld(screen, a.phase, 0, 0, 0, a.clock, 0, 0, 0, false, ridePose{})
		vector.FillRect(screen, 0, 0, ScreenW, ScreenH, color.RGBA{6, 3, 16, 145}, false)
	}
	switch a.scene {
	case sceneName:
		a.drawName(screen)
	case sceneMenu:
		a.drawHome(screen)
	case sceneCourse:
		drawNeonTitle(screen, "Pick a course", ScreenW/2, 72, 42)
		drawLabel(screen, "What do you want to practice?", ScreenW/2, 128, 20, colMuted, text.AlignCenter)
	case sceneBand:
		drawNeonTitle(screen, a.op.Name(), ScreenW/2, 64, 42)
		drawLabel(screen, "Five problems to a gate.", ScreenW/2, 118, 20, colMuted, text.AlignCenter)
		drawLabel(screen, "A few saved seconds come with you. Each gate starts shorter.", ScreenW/2, 146, 20, colMuted, text.AlignCenter)
	case sceneScores:
		a.drawScores(screen)
	case sceneHall:
		a.drawHall(screen)
	case sceneRacers:
		drawNeonTitle(screen, "Racers", ScreenW/2, 72, 42)
	case sceneCountdown:
		a.drawCountdown(screen)
	case sceneResult:
		a.drawResult(screen)
	}
	if a.scene != sceneRace && a.scene != sceneCountdown {
		for i, b := range a.buttons {
			hot := i == a.cursor || b.contains(mx, my)
			drawBtn(screen, b, hot)
		}
	}
}

func (a *App) drawRally(screen *ebiten.Image) {
	shake := 0.0
	if a.shake > 0 {
		shake = math.Sin(a.clock*55) * a.shake * 10
	}
	near := 0.0
	gate := 1
	showGate := false
	if a.race != nil {
		near = a.race.Nearness()
		gate = a.race.Gates() + 1
		showGate, _ = gateDepth(a.race.LegSolved())
	}
	drawWorld(screen, a.phase, near, a.boost, a.burst, a.clock, shake, gate, a.gateDist, showGate, a.ridePose())
	for _, f := range a.floaters {
		a.drawFloater(screen, f, float32(shake))
	}
	if a.race != nil {
		a.drawHUD(screen)
	}
}

func (a *App) drawFloater(screen *ebiten.Image, f floater, shake float32) {
	alpha := f.life / f.max
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	col := color.RGBA{180, 250, 255, uint8(255 * alpha)}
	shadow := color.RGBA{0, 0, 0, uint8(190 * alpha)}
	drawLabel(screen, f.text, f.x+shake+3, f.y+4, f.size, shadow, text.AlignCenter)
	drawLabel(screen, f.text, f.x+shake, f.y, f.size, col, text.AlignCenter)
}

func (a *App) drawHUD(screen *ebiten.Image) {
	for i := 0; i < 9; i++ {
		fade := uint8(168 - i*18)
		vector.FillRect(screen, 0, float32(i*20), ScreenW, 20, color.RGBA{6, 3, 16, fade}, false)
	}

	course := a.op.Name() + "   ·   " + a.band.Name()
	drawLabel(screen, course, 24, 12, 18, colInk, text.AlignStart)
	drawLabel(screen, "Score "+commas(a.race.Score()), 24, 34, 18, colCyan, text.AlignStart)
	drawLabel(screen, fmt.Sprintf("Gate %d", a.race.Gates()+1), ScreenW/2, 16, 22, colAccent, text.AlignCenter)
	if a.scene == sceneRace {
		drawLabel(screen, "backspace undoes a digit", ScreenW-24, 12, 15, colMuted, text.AlignEnd)
		drawLabel(screen, "esc pulls off", ScreenW-24, 32, 15, colMuted, text.AlignEnd)
	}

	a.drawEquation(screen)
	a.drawPips(screen)
	if a.scene != sceneResult {
		a.drawClock(screen)
	}
}

func (a *App) drawEquation(screen *ebiten.Image) {
	p := a.race.Problem()
	prompt := p.Prompt()
	digits := p.AnswerDigits()
	typed := a.race.Typed()
	size := 64.0
	var pw, ph float64
	var boxW, boxH, gap float32
	var total float32
	for size >= 42 {
		pw, ph = text.Measure(prompt, face(size), 0)
		boxH = float32(ph)
		boxW = float32(math.Max(ph*0.78, 52))
		gap = 10
		boxes := float32(len(digits)) * (boxW + gap)
		if boxes > 0 {
			boxes -= gap
		}
		total = float32(pw) + 18 + boxes
		if total <= ScreenW-48 {
			break
		}
		size -= 4
	}
	x := (float32(ScreenW) - total) / 2
	textY := float32(62)
	drawLabel(screen, prompt, x, textY, float32(size), colInk, text.AlignStart)

	wrong := a.clock < a.wrongUntil
	bx := x + float32(pw) + 18
	boxY := textY - 4
	boxH += 8
	for i := range digits {
		stroke := withA(colCyan, 120)
		fill := color.RGBA{10, 6, 22, 230}
		if i < len(typed) {
			stroke = colAccent
		}
		if i == len(typed) {
			stroke = colCyan
		}
		if wrong {
			stroke = colRed
		}
		vector.FillRect(screen, bx, boxY, boxW, boxH, fill, true)
		vector.StrokeRect(screen, bx+1.5, boxY+1.5, boxW-3, boxH-3, 3, stroke, true)
		if i < len(typed) {
			drawLabel(screen, typed[i:i+1], bx+boxW/2, textY, float32(size), colInk, text.AlignCenter)
		} else if i == len(typed) && int(a.clock*2)%2 == 0 {
			vector.FillRect(screen, bx+boxW/2-1.5, boxY+12, 3, boxH-24, colAccent, false)
		}
		bx += boxW + gap
	}
}

func (a *App) drawPips(screen *ebiten.Image) {
	const n = engine.ProblemsPerLeg
	gap := float32(22)
	total := float32(n-1) * gap
	x := float32(ScreenW)/2 - total/2
	y := float32(140)
	for i := 0; i < n; i++ {
		col := withA(colMuted, 180)
		if i < a.race.LegSolved() {
			col = colCyan
		}
		vector.FillCircle(screen, x+float32(i)*gap, y, 6, col, true)
	}
}

func (a *App) drawClock(screen *ebiten.Image) {
	ratio := 1.0
	if a.race.LegClock() > 0 {
		ratio = float64(a.race.TimeLeft()) / float64(a.race.LegClock())
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	barX := float32(360)
	barW := float32(560)
	y := float32(158)
	vector.FillRect(screen, barX, y, barW, 8, color.RGBA{16, 8, 32, 220}, false)
	col := colCyan
	label := colInk
	if ratio < 0.28 {
		col = colRed
		label = colRed
	}
	vector.FillRect(screen, barX, y, barW*float32(ratio), 8, col, false)
	secs := formatSecs(a.race.TimeLeft())
	drawLabel(screen, secs, barX+barW+12, y-8, 20, label, text.AlignStart)
}

func (a *App) drawHome(screen *ebiten.Image) {
	drawNeonTitle(screen, meta.Title, ScreenW/2, 78, 68)
	drawLabel(screen, "Math facts. Reach the gate.", ScreenW/2, 158, 22, colMuted, text.AlignCenter)
	if name := a.racerName(); name != "" {
		drawLabel(screen, name, ScreenW/2, 190, 22, colCyan, text.AlignCenter)
	}
	if a.music != nil {
		line := "M quiets the music"
		if a.music.muted {
			line = "M brings the music back"
		}
		drawLabel(screen, line, ScreenW/2, ScreenH-58, 14, colMuted, text.AlignCenter)
	}
	drawLabel(screen, meta.Copyright, ScreenW/2, ScreenH-36, 14, colMuted, text.AlignCenter)
}

func (a *App) drawName(screen *ebiten.Image) {
	drawNeonTitle(screen, "Who is racing?", ScreenW/2, 110, 46)
	drawLabel(screen, "Type a name, then Enter", ScreenW/2, 168, 20, colMuted, text.AlignCenter)
	x, y, w, h := float32(390), float32(230), float32(500), float32(68)
	vector.FillRect(screen, x, y, w, h, colBtn, true)
	vector.StrokeRect(screen, x+2, y+2, w-4, h-4, 2, colAccent, true)
	shown := string(a.name)
	drawLabel(screen, shown, x+18, y+16, 32, colInk, text.AlignStart)
	if int(a.clock*2)%2 == 0 {
		tw, _ := text.Measure(shown, face(32), 0)
		vector.FillRect(screen, x+20+float32(tw), y+16, 3, 36, colAccent, false)
	}
}

func (a *App) drawScores(screen *ebiten.Image) {
	drawNeonTitle(screen, "Best scores", ScreenW/2, 48, 40)
	who := a.racerName()
	if who == "" {
		who = "No racer yet"
	}
	drawLabel(screen, who, ScreenW/2, 100, 20, colAccent, text.AlignCenter)
	xs := []float32{560, 780, 1000}
	for i, band := range engine.Bands() {
		drawLabel(screen, band.Name(), xs[i], 156, 16, colMuted, text.AlignCenter)
	}
	for i, op := range engine.Courses() {
		y := float32(204 + i*62)
		drawLabel(screen, op.Name(), 80, y, 22, colInk, text.AlignStart)
		for j, band := range engine.Bands() {
			rec := a.store.Best(a.racer, engine.CourseKey(op, band))
			cell := "·"
			col := colMuted
			if rec.Score > 0 {
				cell = commas(rec.Score)
				col = colInk
			}
			drawLabel(screen, cell, xs[j], y, 22, col, text.AlignCenter)
		}
	}
}

func (a *App) drawHall(screen *ebiten.Image) {
	drawNeonTitle(screen, "Hall of Fame", ScreenW/2, 48, 40)
	drawLabel(screen, "Every racer on this computer.", ScreenW/2, 100, 20, colMuted, text.AlignCenter)
	rows := a.store.HallOfFame()
	if len(rows) == 0 {
		drawLabel(screen, "No racers yet.", ScreenW/2, 220, 22, colMuted, text.AlignCenter)
		return
	}
	const (
		markX   = float32(96)
		nameX   = float32(132)
		bestX   = float32(560)
		gatesX  = float32(720)
		racesX  = float32(860)
		courseX = float32(980)
	)
	hy := float32(150)
	drawLabel(screen, "Name", nameX, hy, 16, colMuted, text.AlignStart)
	drawLabel(screen, "Best", bestX, hy, 16, colMuted, text.AlignCenter)
	drawLabel(screen, "Gates", gatesX, hy, 16, colMuted, text.AlignCenter)
	drawLabel(screen, "Races", racesX, hy, 16, colMuted, text.AlignCenter)
	drawLabel(screen, "Course", courseX, hy, 16, colMuted, text.AlignStart)

	gap := float32(40)
	room := float32(590) - float32(188)
	if float32(len(rows))*gap > room {
		gap = room / float32(len(rows))
	}
	size := float32(22)
	if gap < 30 {
		size = 16
	}
	for i, row := range rows {
		y := float32(188) + float32(i)*gap
		col := colInk
		if row.ID == a.racer {
			col = colAccent
			drawLabel(screen, ">", markX, y, size, colAccent, text.AlignStart)
		}
		drawLabel(screen, row.Name, nameX, y, size, col, text.AlignStart)
		best, gates, course := "·", "·", ""
		if row.BestScore > 0 {
			best = commas(row.BestScore)
			gates = fmt.Sprintf("%d", row.BestGates)
			course = courseLabel(row.BestKey)
		}
		drawLabel(screen, best, bestX, y, size, col, text.AlignCenter)
		drawLabel(screen, gates, gatesX, y, size, col, text.AlignCenter)
		drawLabel(screen, fmt.Sprintf("%d", row.Races), racesX, y, size, col, text.AlignCenter)
		drawLabel(screen, course, courseX, y, size, col, text.AlignStart)
	}
}

func courseLabel(key string) string {
	op, band, ok := strings.Cut(key, ":")
	if !ok {
		return key
	}
	var opName, bandName string
	switch op {
	case "add":
		opName = "Addition"
	case "sub":
		opName = "Subtraction"
	case "mul":
		opName = "Multiplication"
	case "div":
		opName = "Division"
	case "mixed":
		opName = "Mixed"
	default:
		return key
	}
	switch band {
	case "easy":
		bandName = "Easy"
	case "medium":
		bandName = "Medium"
	case "hard":
		bandName = "Hard"
	default:
		return key
	}
	return opName + " · " + bandName
}

func (a *App) drawCountdown(screen *ebiten.Image) {
	label := "GO"
	switch {
	case a.count > 1.8:
		label = "3"
	case a.count > 1.2:
		label = "2"
	case a.count > 0.6:
		label = "1"
	}
	cx := float32(ScreenW / 2)
	cy := float32(horizon + 70)
	for _, o := range []float32{-3, 3} {
		drawLabel(screen, label, cx+o, cy+o, 120, color.RGBA{0, 0, 0, 180}, text.AlignCenter)
	}
	drawLabel(screen, label, cx, cy, 120, color.RGBA{255, 244, 220, 255}, text.AlignCenter)
}

func (a *App) drawResult(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, ScreenW, ScreenH, color.RGBA{0, 0, 0, 150}, false)
	x, y, w, h := float32(320), float32(44), float32(640), float32(470)
	vector.FillRect(screen, x, y, w, h, colHUD, true)
	vector.FillRect(screen, x, y, w, 6, colAccent, false)
	title := "The gate closed"
	if a.bailed {
		title = "You pulled off"
	}
	drawLabel(screen, title, ScreenW/2, y+36, 36, colInk, text.AlignCenter)
	drawLabel(screen, a.op.Name()+"   ·   "+a.band.Name(), ScreenW/2, y+88, 18, colMuted, text.AlignCenter)
	drawLabel(screen, commas(a.race.Score()), ScreenW/2, y+124, 64, colAccent, text.AlignCenter)
	line := "The first gate is still ahead."
	switch a.race.Gates() {
	case 1:
		line = "You cleared 1 gate."
	default:
		if a.race.Gates() > 1 {
			line = fmt.Sprintf("You cleared %d gates.", a.race.Gates())
		}
	}
	drawLabel(screen, line, ScreenW/2, y+210, 22, colInk, text.AlignCenter)
	if a.newBest {
		drawLabel(screen, "New best", ScreenW/2, y+248, 22, colAccent, text.AlignCenter)
	} else if rec := a.store.Best(a.racer, engine.CourseKey(a.op, a.band)); rec.Score > 0 {
		drawLabel(screen, "Best "+commas(rec.Score), ScreenW/2, y+248, 22, colMuted, text.AlignCenter)
	}
}

func (a *App) updateName(chars []rune) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.back()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		a.confirmName()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(a.name) > 0 {
		a.name = a.name[:len(a.name)-1]
	}
	for _, r := range chars {
		if len(a.name) >= 16 {
			break
		}
		if r == ' ' || r == '-' || r == '\'' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			a.name = append(a.name, r)
		}
	}
}

func (a *App) updatePick(clicked bool, mx, my int) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.back()
		return
	}
	n := len(a.buttons)
	if n == 0 {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		a.cursor = (a.cursor + 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		a.cursor = (a.cursor - 1 + n) % n
	}
	if a.scene == sceneCourse {
		if d, ok := pressedDigit(); ok && d >= 1 && d < n {
			a.activate(a.buttons[d-1].id)
			return
		}
	}
	if a.scene == sceneBand {
		if d, ok := pressedDigit(); ok && d >= 1 && d <= 3 && d < n {
			a.activate(a.buttons[d-1].id)
			return
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		if a.cursor >= 0 && a.cursor < n {
			a.activate(a.buttons[a.cursor].id)
		}
		return
	}
	a.clickButtons(clicked, mx, my)
}

func (a *App) updateCountdown(dt time.Duration) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.setScene(sceneBand)
		return
	}
	a.advanceRoad(dt.Seconds(), false)
	a.count -= dt.Seconds()
	if a.count <= 0 {
		a.scene = sceneRace
		a.buttons = nil
	}
}

func (a *App) updateRace(dt time.Duration) {
	if a.pulling {
		a.pullAge += dt.Seconds()
		a.advanceRoad(dt.Seconds(), false)
		if a.pullAge >= pullDur || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.pullAge = pullDur
			a.endRace()
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.bailed = true
		a.pulling = true
		a.pullFrom = a.lane
		a.pullAge = 0
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		a.race.Backspace()
	}
	if d, ok := pressedDigit(); ok {
		a.react(a.race.TypeDigit(d))
		if a.race.Over() {
			a.bailed = false
			a.endRace()
			return
		}
	}
	a.react(a.race.Tick(dt))
	if a.race.Over() {
		a.bailed = false
		a.endRace()
		return
	}
	a.advanceRoad(dt.Seconds(), true)
}

func (a *App) react(res engine.Result) {
	switch res.Kind {
	case engine.KindCorrect:
		if a.boost < 1 {
			a.boost = 1
		}
	case engine.KindWrong:
		a.shake = 1
		a.wrongUntil = a.clock + 0.32
		if a.bump == 0 {
			a.bump = 1
		}
		a.lane = float64(a.bump)
		a.bump = -a.bump
		a.sparks = 1
	case engine.KindGate:
		a.boost = 1
		a.burst = 1
		a.spawn("+" + commas(res.Points))
	}
}

func (a *App) spawn(text string) {
	a.floaters = append(a.floaters, floater{
		text: text,
		x:    ScreenW / 2,
		y:    450,
		life: 0.95,
		max:  0.95,
		size: 52,
	})
}

func (a *App) decay(dt float64) {
	if a.boost > 0 {
		a.boost -= dt / 0.45
		if a.boost < 0 {
			a.boost = 0
		}
	}
	if a.shake > 0 {
		a.shake -= dt / 0.28
		if a.shake < 0 {
			a.shake = 0
		}
	}
	if a.burst > 0 {
		a.burst -= dt / 0.55
		if a.burst < 0 {
			a.burst = 0
		}
	}
	if a.sparks > 0 {
		a.sparks -= dt / 0.45
		if a.sparks < 0 {
			a.sparks = 0
		}
	}
	if !a.pulling {
		a.lane += (0 - a.lane) * dt * 2.4
	}
	kept := a.floaters[:0]
	for _, f := range a.floaters {
		f.life -= dt
		f.y -= float32(64 * dt)
		if f.life > 0 {
			kept = append(kept, f)
		}
	}
	a.floaters = kept
}

func (a *App) roadSpeed(racing bool) float64 {
	near := 0.0
	if a.race != nil {
		near = a.race.Nearness()
	}
	hurry := 0.0
	if near > 0.72 {
		hurry = (near - 0.72) / 0.28
	}
	speed := 0.18
	if racing {
		speed = 0.5 + a.boost*1.7 + hurry*0.7
	}
	if a.pulling {
		left := 1 - a.pullAge/pullDur
		if left < 0 {
			left = 0
		}
		speed = 0.9 * left
	}
	return speed
}

func (a *App) advanceRoad(dt float64, racing bool) {
	a.phase += a.roadSpeed(racing) * dt
}

// approachGate eases the gate toward the step for this problem. The first
// problem keeps it hidden. A gate flash holds the last distance until the
// burst fades, so the flash stays where the gate was.
func (a *App) approachGate(dt float64) {
	if a.race == nil {
		return
	}
	show, want := gateDepth(a.race.LegSolved())
	if !show {
		if a.burst <= 0 {
			a.gateDist = 0
		}
		return
	}
	if a.gateDist <= 0.001 {
		a.gateDist = want
		return
	}
	a.gateDist += (want - a.gateDist) * math.Min(1, dt*8)
}

func (a *App) syncEngine() {
	on := a.scene == sceneCountdown || a.scene == sceneRace
	a.music.hear(a.roadSpeed(a.scene == sceneRace), a.boost, a.sparks, on)
}

func (a *App) ridePose() ridePose {
	lane := a.lane
	lean := a.lane * 0.55
	u := 0.0
	if a.pulling {
		u = a.pullAge / pullDur
		if u > 1 {
			u = 1
		}
		e := smooth(u)
		lane = a.pullFrom*(1-e) + 1.18*e
		lean = math.Sin(u*math.Pi) * 0.9
	} else {
		lean += bendLean(a.phase)
	}
	park := a.pulling && u > 0.82
	return ridePose{
		show:    true,
		lane:    lane,
		wheelie: a.boost,
		lean:    lean,
		sparks:  a.sparks,
		park:    park,
	}
}

func (a *App) endRace() {
	if a.scene == sceneResult || a.race == nil {
		return
	}
	key := engine.CourseKey(a.op, a.band)
	prev := a.store.Best(a.racer, key)
	a.newBest = false
	a.store.AddRace(a.racer)
	if a.race.Score() > prev.Score {
		a.newBest = true
		a.store.SetBest(a.racer, key, profile.Record{Score: a.race.Score(), Gates: a.race.Gates()})
	}
	_ = a.store.Save()
	a.scene = sceneResult
	a.cursor = 0
	a.buttons = a.layout(sceneResult)
}

func (a *App) startCountdown() {
	a.race = engine.New(engine.Config{Op: a.op, Band: a.band}, nil)
	a.bailed = false
	a.newBest = false
	a.floaters = nil
	a.boost = 0
	a.shake = 0
	a.burst = 0
	a.gateDist = 0
	a.wrongUntil = 0
	a.lane = 0
	a.pullFrom = 0
	a.pullAge = 0
	a.sparks = 0
	a.bump = 1
	a.pulling = false
	a.count = 2.4
	a.phase = 0
	a.scene = sceneCountdown
	a.buttons = nil
	a.cursor = 0
}

func (a *App) confirmName() {
	a.racer = a.store.Upsert(string(a.name))
	if a.racer == "" {
		return
	}
	_ = a.store.Save()
	a.name = nil
	a.setScene(sceneMenu)
}

func (a *App) back() {
	switch a.scene {
	case sceneCourse, sceneScores, sceneHall, sceneResult:
		a.setScene(sceneMenu)
	case sceneBand:
		a.setScene(sceneCourse)
	case sceneRacers:
		a.setScene(sceneMenu)
	case sceneName:
		if len(a.store.Profiles) > 0 {
			a.setScene(sceneRacers)
		}
	}
}

func (a *App) activate(id string) {
	switch id {
	case "quit":
		a.wantQuit = true
	case "race":
		a.setScene(sceneCourse)
	case "scores":
		a.setScene(sceneScores)
	case "hall":
		a.setScene(sceneHall)
	case "racers":
		a.setScene(sceneRacers)
	case "back":
		a.back()
	case "new":
		a.name = nil
		a.setScene(sceneName)
	case "start":
		a.confirmName()
	case "add":
		a.pickCourse(engine.OpAdd)
	case "sub":
		a.pickCourse(engine.OpSub)
	case "mul":
		a.pickCourse(engine.OpMul)
	case "div":
		a.pickCourse(engine.OpDiv)
	case "mixed":
		a.pickCourse(engine.OpMixed)
	case "easy":
		a.band = engine.BandEasy
		a.startCountdown()
	case "medium":
		a.band = engine.BandMedium
		a.startCountdown()
	case "hard":
		a.band = engine.BandHard
		a.startCountdown()
	case "again":
		a.startCountdown()
	case "menu":
		a.setScene(sceneMenu)
	default:
		if strings.HasPrefix(id, "racer:") {
			if a.store.Select(id[len("racer:"):]) {
				a.racer = a.store.Active
				_ = a.store.Save()
				a.setScene(sceneMenu)
			}
		}
	}
}

func (a *App) pickCourse(op engine.Op) {
	a.op = op
	a.setScene(sceneBand)
}

func (a *App) setScene(s scene) {
	a.scene = s
	a.cursor = 0
	a.buttons = a.layout(s)
}

func (a *App) layout(s scene) []button {
	switch s {
	case sceneMenu:
		return column([]item{
			{"Race", "", "race"},
			{"Scores", "", "scores"},
			{"Hall of Fame", "", "hall"},
			{"Switch racer", "", "racers"},
			{"Quit", "", "quit"},
		}, 258, 52, 8, 400)
	case sceneName:
		items := []item{{"Start", "", "start"}}
		if len(a.store.Profiles) > 0 {
			items = append(items, item{"Back", "", "back"})
		}
		return column(items, 340, 52, 12, 320)
	case sceneCourse:
		items := make([]item, 0, 6)
		for _, op := range engine.Courses() {
			items = append(items, item{op.Name(), "", op.Key()})
		}
		items = append(items, item{"Back", "", "back"})
		return column(items, 176, 52, 8, 440)
	case sceneBand:
		items := make([]item, 0, 4)
		for _, band := range engine.Bands() {
			items = append(items, item{band.Name(), bandBlurb(a.op, band), band.Key()})
		}
		btns := column(items, 190, 74, 10, 560)
		back := column([]item{{"Back", "", "back"}}, 190+3*84, 52, 0, 560)
		return append(btns, back...)
	case sceneScores, sceneHall:
		return column([]item{{"Back", "", "back"}}, 620, 52, 0, 280)
	case sceneRacers:
		items := make([]item, 0, len(a.store.Profiles)+2)
		for _, p := range a.store.Profiles {
			label := p.Name
			if p.ID == a.racer {
				label = p.Name + "   ·   racing"
			}
			items = append(items, item{label, "", "racer:" + p.ID})
		}
		items = append(items, item{"New racer", "", "new"}, item{"Back", "", "back"})
		y := float32(140)
		h := float32(48)
		gap := float32(8)
		if len(items) > 1 {
			room := float32(ScreenH-36) - y
			total := float32(len(items))*h + float32(len(items)-1)*gap
			if total > room {
				gap = 6
				h = (room - gap*float32(len(items)-1)) / float32(len(items))
			}
		}
		return column(items, y, h, gap, 440)
	case sceneResult:
		return []button{
			{x: 372, y: 400, w: 240, h: 52, label: "Race again", id: "again"},
			{x: 668, y: 400, w: 240, h: 52, label: "Menu", id: "menu"},
		}
	default:
		return nil
	}
}

func (a *App) racerName() string {
	if a.store == nil {
		return ""
	}
	return a.store.Name(a.racer)
}

func column(items []item, y, h, gap, w float32) []button {
	x := (float32(ScreenW) - w) / 2
	out := make([]button, len(items))
	for i, it := range items {
		out[i] = button{
			x: x, y: y + float32(i)*(h+gap), w: w, h: h,
			label: it.label, sub: it.sub, id: it.id,
		}
	}
	return out
}

func bandBlurb(op engine.Op, b engine.Band) string {
	switch op {
	case engine.OpAdd:
		switch b {
		case engine.BandEasy:
			return "Sums up to 10"
		case engine.BandMedium:
			return "Sums from 11 to 20"
		default:
			return "Two-digit plus two-digit"
		}
	case engine.OpSub:
		switch b {
		case engine.BandEasy:
			return "Within 10"
		case engine.BandMedium:
			return "Within 20"
		default:
			return "Two-digit minus"
		}
	case engine.OpMul:
		switch b {
		case engine.BandEasy:
			return "Tables 0 through 5"
		case engine.BandMedium:
			return "Tables 0 through 10"
		default:
			return "11s, 12s, and two-digit times one-digit"
		}
	case engine.OpDiv:
		switch b {
		case engine.BandEasy:
			return "Exact, tables 0 through 5"
		case engine.BandMedium:
			return "Exact, tables 0 through 10"
		default:
			return "Exact, bigger tables and two-digit answers"
		}
	default:
		switch b {
		case engine.BandEasy:
			return "All four operations, easy facts"
		case engine.BandMedium:
			return "All four operations, medium facts"
		default:
			return "All four operations, hard facts"
		}
	}
}

func (a *App) followMouse(mx, my int) {
	if mx == a.mx && my == a.my {
		return
	}
	a.mx, a.my = mx, my
	for i, b := range a.buttons {
		if b.contains(mx, my) {
			a.cursor = i
			return
		}
	}
}

func (a *App) clickButtons(clicked bool, mx, my int) {
	if !clicked {
		return
	}
	for _, b := range a.buttons {
		if b.contains(mx, my) {
			a.activate(b.id)
			return
		}
	}
}

func pressedDigit() (int, bool) {
	tops := []ebiten.Key{
		ebiten.KeyDigit0, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4,
		ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9,
	}
	pads := []ebiten.Key{
		ebiten.KeyNumpad0, ebiten.KeyNumpad1, ebiten.KeyNumpad2, ebiten.KeyNumpad3, ebiten.KeyNumpad4,
		ebiten.KeyNumpad5, ebiten.KeyNumpad6, ebiten.KeyNumpad7, ebiten.KeyNumpad8, ebiten.KeyNumpad9,
	}
	for i, k := range tops {
		if inpututil.IsKeyJustPressed(k) {
			return i, true
		}
	}
	for i, k := range pads {
		if inpututil.IsKeyJustPressed(k) {
			return i, true
		}
	}
	return 0, false
}

func frameDT() time.Duration {
	tps := ebiten.ActualTPS()
	dt := time.Second / 60
	if tps >= 30 && tps <= 240 {
		dt = time.Duration(float64(time.Second) / tps)
	}
	if dt > 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	if dt < 0 {
		return time.Second / 60
	}
	return dt
}

func formatSecs(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := d.Milliseconds()
	return fmt.Sprintf("%d.%ds", ms/1000, (ms%1000)/100)
}
