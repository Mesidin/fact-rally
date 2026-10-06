package ui

import (
	"math"
	"testing"
)

func TestCurveStaysUnderTheRider(t *testing.T) {
	for _, phase := range []float64{0, 0.4, 1.2, 2.5, 4} {
		x, _ := roadX(1, 0, 0, phase)
		if d := math.Abs(float64(x) - float64(ScreenW)/2); d > 6 {
			t.Fatalf("phase %v shifted the lane under the rider by %v", phase, d)
		}
	}
}

func TestHorizonSwingsFartherThanTheMiddle(t *testing.T) {
	const a = 0.2
	const b = 2.4
	farA, _ := roadX(0, 0, 0, a)
	farB, _ := roadX(0, 0, 0, b)
	far := math.Abs(float64(farA - farB))
	if far < 80 {
		t.Fatalf("horizon moved %v px between phases", far)
	}
	midA, _ := roadX(0.55, 0, 0, a)
	midB, _ := roadX(0.55, 0, 0, b)
	mid := math.Abs(float64(midA - midB))
	if mid >= far {
		t.Fatalf("mid-road swing %v should stay inside the horizon swing %v", mid, far)
	}
}

func TestRiderLeansIntoTheBend(t *testing.T) {
	// These phases put the horizon hard to one side of the rider.
	left := curveAt(0, 2.96)
	right := curveAt(0, 0.44)
	if left >= -200 {
		t.Fatalf("expected a left bend, horizon offset %v", left)
	}
	if right <= 200 {
		t.Fatalf("expected a right bend, horizon offset %v", right)
	}
	if bendLean(2.96) >= 0 {
		t.Fatalf("left bend leaned %v", bendLean(2.96))
	}
	if bendLean(0.44) <= 0 {
		t.Fatalf("right bend leaned %v", bendLean(0.44))
	}
}

func TestGateHidesThenStepsCloser(t *testing.T) {
	if show, _ := gateDepth(0); show {
		t.Fatal("first problem shows a gate")
	}
	var prev float64
	for solved := 1; solved <= 4; solved++ {
		show, dist := gateDepth(solved)
		if !show {
			t.Fatalf("problem %d hid the gate", solved+1)
		}
		if solved > 1 && dist-prev < 0.12 {
			t.Fatalf("step to solved %d is %v", solved, dist-prev)
		}
		if dist <= prev {
			t.Fatalf("solved %d dist %v", solved, dist)
		}
		prev = dist
	}
	_, far := gateDepth(1)
	_, near := gateDepth(4)
	if far > 0.12 || near < 0.55 || near > 0.8 {
		t.Fatalf("range far %v near %v", far, near)
	}
}

func TestEachGateChangesTheRoadside(t *testing.T) {
	if sloganFor(1) != "Math rocks!" {
		t.Fatal(sloganFor(1))
	}
	if sloganFor(2) == sloganFor(1) || sloganFor(3) == sloganFor(2) {
		t.Fatalf("slogans %q %q %q", sloganFor(1), sloganFor(2), sloganFor(3))
	}
	same := true
	for slot := 0; slot < 8; slot++ {
		if propKindFor(1, slot) != propKindFor(2, slot) {
			same = false
		}
		if propKindFor(slot+1, 3) != propBoard {
			t.Fatalf("gate %d billboard %d", slot+1, propKindFor(slot+1, 3))
		}
	}
	if same {
		t.Fatal("gate 1 and 2 share every prop")
	}
}

func TestLeanSpriteMatchesTheBend(t *testing.T) {
	// The painting puts the rider's body on the viewer's left of the tire.
	// A left bend uses it as-is. A right bend mirrors it.
	use, flip := leanSprite(bendLean(2.96))
	if !use || flip {
		t.Fatalf("left bend use=%v flip=%v", use, flip)
	}
	use, flip = leanSprite(bendLean(0.44))
	if !use || !flip {
		t.Fatalf("right bend use=%v flip=%v", use, flip)
	}
}
