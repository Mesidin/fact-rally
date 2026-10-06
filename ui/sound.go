package ui

import (
	"encoding/binary"
	"log"
	"math"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// The engine is original synthesis: a high whine that climbs inside each
// gear and drops on the shift, the way an arcade bike sounds when it
// picks up speed. It is not a recording. newPlaylist does not start it.
// hear is a no-op until startEngine is called again.
const (
	gearCount    = 4
	gearLow      = 0.35
	gearHigh     = 2.70
	engineVolume = 0.32
)

// engineTone is the whine and the exhaust pulse for a road speed.
// speed uses the same units as the highway (about 0.2 at idle, past 2 flat out).
// boost is 0..1. gear is 0..3 and frac is how far through that gear the whine has climbed.
func engineTone(speed, boost float64) (whineHz, lowHz float64, gear int, frac float64) {
	if speed < 0 {
		speed = 0
	}
	if boost < 0 {
		boost = 0
	}
	if boost > 1 {
		boost = 1
	}
	lowHz = 42 + speed*18 + boost*10
	if speed < gearLow {
		whineHz = 210 + speed*180 + boost*140
		return whineHz, lowHz, 0, 0
	}
	span := (speed - gearLow) / (gearHigh - gearLow)
	if span > 1 {
		span = 1
	}
	pos := span * float64(gearCount)
	gear = int(pos)
	if gear >= gearCount {
		gear = gearCount - 1
		frac = 1
	} else {
		frac = pos - float64(gear)
	}
	lo := 460 + float64(gear)*120
	hi := lo * 1.85
	s := frac * frac * (3 - 2*frac)
	whineHz = lo + (hi-lo)*s + boost*200
	return whineHz, lowHz, gear, frac
}

// engineVoice is an endless stereo stream. The game sets the targets;
// Read runs on the audio thread.
type engineVoice struct {
	speed   atomic.Uint64
	boost   atomic.Uint64
	scrape  atomic.Uint64
	audible atomic.Uint64

	smSpeed  float64
	smBoost  float64
	smScrape float64
	smGain   float64
	whineHz  float64
	phWhine  float64
	phLow    float64
	gear     int
	tick     int
	rng      uint32
}

func (e *engineVoice) set(speed, boost, scrape float64, on bool) {
	if e == nil {
		return
	}
	e.speed.Store(math.Float64bits(speed))
	e.boost.Store(math.Float64bits(boost))
	e.scrape.Store(math.Float64bits(scrape))
	if on {
		e.audible.Store(1)
	} else {
		e.audible.Store(0)
	}
}

func loadBits(a *atomic.Uint64) float64 {
	return math.Float64frombits(a.Load())
}

func (e *engineVoice) Read(p []byte) (int, error) {
	if e.rng == 0 {
		e.rng = 0xA341316C
	}
	if e.whineHz == 0 {
		e.whineHz = 220
	}
	frames := len(p) / 8
	for i := frames * 8; i < len(p); i++ {
		p[i] = 0
	}
	targetS := loadBits(&e.speed)
	targetB := loadBits(&e.boost)
	targetC := loadBits(&e.scrape)
	want := 0.0
	if e.audible.Load() != 0 {
		want = 1
	}
	const rate = 44100.0
	for i := 0; i < frames; i++ {
		e.smSpeed += (targetS - e.smSpeed) * 0.00028
		e.smBoost += (targetB - e.smBoost) * 0.0006
		e.smScrape += (targetC - e.smScrape) * 0.002
		e.smGain += (want - e.smGain) * 0.0005

		hz, low, gear, _ := engineTone(e.smSpeed, e.smBoost)
		hz *= 1 - 0.28*e.smScrape
		e.whineHz += (hz - e.whineHz) * 0.003
		if gear != e.gear {
			e.gear = gear
			e.tick = 650
		}
		e.phWhine += e.whineHz / rate
		e.phLow += low / rate
		if e.phWhine >= 1 {
			e.phWhine -= math.Floor(e.phWhine)
		}
		if e.phLow >= 1 {
			e.phLow -= math.Floor(e.phLow)
		}

		buzz := pulse(e.phWhine, 0.32)*0.62 + pulse(e.phWhine*2.005, 0.5)*0.38
		thump := pulse(e.phLow, 0.42)
		n := e.noise()
		whineAmp := 0.13 + 0.10*clamp01(e.smSpeed/1.4) + 0.06*clamp01(e.smBoost)
		whineAmp *= 1 - 0.5*clamp01(e.smScrape)
		lowAmp := 0.07 + 0.05*(1-clamp01(e.smSpeed/0.7))
		windAmp := 0.012 + 0.018*clamp01(e.smSpeed/2) + 0.02*clamp01(e.smBoost)
		tickAmp := 0.0
		if e.tick > 0 {
			tickAmp = 0.16 * float64(e.tick) / 650
			e.tick--
		}
		sample := buzz*whineAmp + thump*lowAmp + n*windAmp + n*e.smScrape*0.4 + n*tickAmp
		sample *= e.smGain
		if sample > 0.9 {
			sample = 0.9
		}
		if sample < -0.9 {
			sample = -0.9
		}
		bits := math.Float32bits(float32(sample))
		binary.LittleEndian.PutUint32(p[i*8:], bits)
		binary.LittleEndian.PutUint32(p[i*8+4:], bits)
	}
	return frames * 8, nil
}

func (e *engineVoice) noise() float64 {
	x := e.rng
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	e.rng = x
	return float64(int32(x)) / float64(math.MaxInt32)
}

func pulse(phase, duty float64) float64 {
	if phase-math.Floor(phase) < duty {
		return 1
	}
	return -1
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func startEngine(ctx *audio.Context) (*engineVoice, *audio.Player) {
	e := &engineVoice{whineHz: 220}
	p, err := ctx.NewPlayerF32(e)
	if err != nil {
		log.Printf("engine: %v", err)
		return nil, nil
	}
	p.SetVolume(engineVolume)
	p.Play()
	return e, p
}
