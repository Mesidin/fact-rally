package ui

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestWhineRisesInsideAGearAndDropsOnTheShift(t *testing.T) {
	prevHz := 0.0
	prevGear := 0
	drops := 0
	for i := 0; i <= 400; i++ {
		s := gearLow + float64(i)*(gearHigh-gearLow)/400
		hz, _, gear, _ := engineTone(s, 0)
		if hz <= 0 || math.IsNaN(hz) {
			t.Fatalf("speed %.3f hz %v", s, hz)
		}
		if i > 0 && gear != prevGear {
			if hz >= prevHz {
				t.Fatalf("gear %d to %d at speed %.3f did not drop: %.1f -> %.1f", prevGear, gear, s, prevHz, hz)
			}
			drops++
		}
		if i > 0 && gear == prevGear && hz+0.01 < prevHz {
			t.Fatalf("pitch fell inside gear %d at speed %.3f: %.1f -> %.1f", gear, s, prevHz, hz)
		}
		prevHz = hz
		prevGear = gear
	}
	if drops != gearCount-1 {
		t.Fatalf("shifts = %d, want %d", drops, gearCount-1)
	}
}

func TestBoostRaisesTheWhine(t *testing.T) {
	idle, _, _, _ := engineTone(1.2, 0)
	full, _, _, _ := engineTone(1.2, 1)
	if full < idle+100 {
		t.Fatalf("boost %v idle %v", full, idle)
	}
}

func TestEngineStaysQuietUntilTheRace(t *testing.T) {
	e := &engineVoice{}
	e.set(1.8, 1, 0, false)
	buf := make([]byte, 44100/5*8)
	n, err := e.Read(buf)
	if err != nil || n != len(buf) {
		t.Fatalf("read %d %v", n, err)
	}
	if peak := pcmPeak(buf[:n]); peak > 0.002 {
		t.Fatalf("silent engine peaked at %v", peak)
	}
}

func TestEngineReadStaysInRange(t *testing.T) {
	e := &engineVoice{}
	e.set(1.6, 0.8, 0.2, true)
	buf := make([]byte, 44100/2*8)
	n, err := e.Read(buf)
	if err != nil || n != len(buf) {
		t.Fatalf("read %d %v", n, err)
	}
	peak := pcmPeak(buf[:n])
	if peak > 0.91 {
		t.Fatalf("peak %v", peak)
	}
	if peak < 0.05 {
		t.Fatalf("too quiet %v", peak)
	}
	// Stereo pair matches. The bike is centered.
	left := math.Float32frombits(binary.LittleEndian.Uint32(buf[n-8:]))
	right := math.Float32frombits(binary.LittleEndian.Uint32(buf[n-4:]))
	if left != right {
		t.Fatalf("stereo %v %v", left, right)
	}
}

func pcmPeak(buf []byte) float32 {
	var peak float32
	for i := 0; i+4 <= len(buf); i += 4 {
		f := math.Float32frombits(binary.LittleEndian.Uint32(buf[i:]))
		if f < 0 {
			f = -f
		}
		if f > peak {
			peak = f
		}
	}
	return peak
}
