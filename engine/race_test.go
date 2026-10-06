package engine

import (
	"testing"
	"time"
)

type script struct {
	ps []Problem
	i  int
}

func (s *script) Next() Problem {
	p := s.ps[s.i]
	if s.i+1 < len(s.ps) {
		s.i++
	}
	return p
}

func fact(left, right int, op Op, answer int) Problem {
	return Problem{Left: left, Right: right, Op: op, Answer: answer}
}

func TestMultidigitSubmitsOnLastDigit(t *testing.T) {
	src := &script{ps: []Problem{
		fact(8, 7, OpMul, 56),
		fact(3, 4, OpAdd, 7),
	}}
	s := New(Config{Op: OpMul, Band: BandEasy}, src)
	if s.Problem().Answer != 56 {
		t.Fatalf("start %+v", s.Problem())
	}
	if got := s.TypeDigit(5); got.Kind != KindNone || s.Typed() != "5" || s.Problem().Answer != 56 {
		t.Fatalf("partial %+v typed %q problem %+v", got, s.Typed(), s.Problem())
	}
	got := s.TypeDigit(6)
	if got.Kind != KindCorrect || s.Problem().Answer != 7 || s.Typed() != "" || s.LegSolved() != 1 {
		t.Fatalf("advance %+v problem %+v typed %q leg %d", got, s.Problem(), s.Typed(), s.LegSolved())
	}
}

func TestWrongClearsAndKeepsProblem(t *testing.T) {
	src := &script{ps: []Problem{fact(8, 7, OpMul, 56)}}
	s := New(Config{Op: OpMul, Band: BandEasy}, src)
	base := s.TimeLeft()
	s.TypeDigit(5)
	s.Backspace()
	if s.Typed() != "" {
		t.Fatalf("backspace %q", s.Typed())
	}
	s.TypeDigit(5)
	got := s.TypeDigit(0)
	if got.Kind != KindWrong || s.Typed() != "" || s.Problem().Answer != 56 {
		t.Fatalf("wrong %+v typed %q problem %+v", got, s.Typed(), s.Problem())
	}
	if s.TimeLeft() != base-WrongPenalty {
		t.Fatalf("penalty %s", s.TimeLeft())
	}
}

func TestCheckpointScoresTimeLeft(t *testing.T) {
	ps := make([]Problem, 6)
	for i := range ps {
		ps[i] = fact(2, 2, OpAdd, 4)
	}
	s := New(Config{Op: OpAdd, Band: BandEasy}, &script{ps: ps})
	base := BaseTime(BandEasy)
	var got Result
	for i := 0; i < ProblemsPerLeg; i++ {
		got = s.TypeDigit(4)
	}
	wantPoints := 5 * int(base.Milliseconds()) / 100
	if got.Kind != KindGate || got.Points != wantPoints || got.Gate != 1 {
		t.Fatalf("gate %+v want points %d", got, wantPoints)
	}
	if s.Score() != wantPoints || s.Gates() != 1 || s.LegSolved() != 0 {
		t.Fatalf("score %d gates %d leg %d", s.Score(), s.Gates(), s.LegSolved())
	}
	if s.TimeLeft() != NextClock(1, base) || s.LegClock() != s.TimeLeft() {
		t.Fatalf("next clock %s", s.TimeLeft())
	}
	if s.TimeLeft() != LegPace(1)+carryCap {
		t.Fatalf("spare over the cap still carried: %s", s.TimeLeft())
	}
	if s.Problem().Answer != 4 || s.Nearness() != 0 {
		t.Fatalf("next problem %+v near %v", s.Problem(), s.Nearness())
	}
}

func TestPhotoFinishScoresAPoint(t *testing.T) {
	ps := make([]Problem, 6)
	for i := range ps {
		ps[i] = fact(1, 1, OpAdd, 2)
	}
	s := New(Config{Op: OpAdd, Band: BandEasy}, &script{ps: ps})
	s.Tick(BaseTime(BandEasy) - time.Millisecond)
	var got Result
	for i := 0; i < ProblemsPerLeg; i++ {
		got = s.TypeDigit(2)
	}
	if got.Kind != KindGate || got.Points != 1 || s.Score() != 1 {
		t.Fatalf("photo %+v score %d", got, s.Score())
	}
}

func TestClockEndsTheRace(t *testing.T) {
	s := New(Config{Op: OpAdd, Band: BandEasy}, &script{ps: []Problem{fact(1, 1, OpAdd, 2)}})
	s.TypeDigit(2)
	if s.Tick(time.Hour).Kind != KindEnded || !s.Over() || s.Score() != 0 || s.Gates() != 0 {
		t.Fatalf("end score %d gates %d over %v", s.Score(), s.Gates(), s.Over())
	}
	if s.Tick(time.Second).Kind != KindNone || s.TypeDigit(2).Kind != KindNone {
		t.Fatal("input after the end")
	}
}

func TestWrongAnswerCanCloseTheGate(t *testing.T) {
	s := New(Config{Op: OpAdd, Band: BandEasy}, &script{ps: []Problem{fact(3, 4, OpAdd, 7)}})
	s.Tick(BaseTime(BandEasy) - 500*time.Millisecond)
	got := s.TypeDigit(8)
	if got.Kind != KindEnded || !s.Over() || s.Score() != 0 {
		t.Fatalf("ended %+v over %v score %d", got, s.Over(), s.Score())
	}
}

func TestSpeedIsTheSameOnEveryBand(t *testing.T) {
	easy, med, hard := BaseTime(BandEasy), BaseTime(BandMedium), BaseTime(BandHard)
	if easy != 40*time.Second || easy != med || med != hard {
		t.Fatalf("bases %s %s %s", easy, med, hard)
	}
}

func TestPaceShrinksTowardOneSecondAProblem(t *testing.T) {
	if LegPace(0) != 40*time.Second {
		t.Fatalf("opening %s", LegPace(0))
	}
	prev := LegPace(0)
	for gates := 1; gates <= 11; gates++ {
		pace := LegPace(gates)
		if pace >= prev {
			t.Fatalf("after %d gates pace %s did not drop from %s", gates, pace, prev)
		}
		prev = pace
	}
	// Eight cleared is the start of gate 9. Eleven cleared is gate 12.
	if LegPace(8) != 10*time.Second {
		t.Fatalf("gate 9 %s", LegPace(8))
	}
	if LegPace(11) != 5*time.Second {
		t.Fatalf("gate 12 %s", LegPace(11))
	}
	if LegPace(16) != 0 || LegPace(40) != 0 {
		t.Fatalf("late %s %s", LegPace(16), LegPace(40))
	}
}

func TestSavedSecondsCarryUpToTheCap(t *testing.T) {
	if NextClock(1, 30*time.Second) != LegPace(1)+carryCap {
		t.Fatalf("bank %s", NextClock(1, 30*time.Second))
	}
	if NextClock(1, 2*time.Second) != LegPace(1)+2*time.Second {
		t.Fatalf("small spare %s", NextClock(1, 2*time.Second))
	}
	if NextClock(1, 2*time.Second) >= 40*time.Second {
		t.Fatalf("scrape grew to %s", NextClock(1, 2*time.Second))
	}
	ps := make([]Problem, 12)
	for i := range ps {
		ps[i] = fact(2, 2, OpAdd, 4)
	}
	s := New(Config{Op: OpAdd, Band: BandHard}, &script{ps: ps})
	for g := 0; g < 2; g++ {
		for n := 0; n < ProblemsPerLeg; n++ {
			s.TypeDigit(4)
		}
	}
	first := LegPace(1) + carryCap
	second := LegPace(2) + carryCap
	if s.Gates() != 2 || s.TimeLeft() != second || first != 40*time.Second {
		t.Fatalf("clock %s gates %d want %s", s.TimeLeft(), s.Gates(), second)
	}
}

func TestThinkingPaceIsOutBeforeGate9(t *testing.T) {
	// Four seconds a problem: time to think, find the key, and type.
	ps := make([]Problem, 80)
	for i := range ps {
		ps[i] = fact(2, 2, OpAdd, 4)
	}
	s := New(Config{Op: OpDiv, Band: BandEasy}, &script{ps: ps})
	for !s.Over() && s.Gates() < 9 {
		if s.Tick(4*time.Second).Kind == KindEnded {
			break
		}
		s.TypeDigit(4)
	}
	if !s.Over() || s.Gates() >= 9 {
		t.Fatalf("gate %d still has %s", s.Gates()+1, s.TimeLeft())
	}
}

func TestQuickPaceDoesNotBankMinutes(t *testing.T) {
	ps := make([]Problem, 80)
	for i := range ps {
		ps[i] = fact(2, 2, OpAdd, 4)
	}
	s := New(Config{Op: OpDiv, Band: BandEasy}, &script{ps: ps})
	for !s.Over() && s.Gates() < 8 {
		if s.Tick(2*time.Second).Kind == KindEnded {
			t.Fatal("two seconds a problem ended before gate 9")
		}
		s.TypeDigit(4)
	}
	if s.Gates() != 8 || s.TimeLeft() > 20*time.Second {
		t.Fatalf("gate 9 clock %s after %d gates", s.TimeLeft(), s.Gates())
	}
}

func TestNearnessTracksTheClock(t *testing.T) {
	s := New(Config{Op: OpAdd, Band: BandEasy}, &script{ps: []Problem{fact(1, 1, OpAdd, 2)}})
	if s.Nearness() != 0 {
		t.Fatal(s.Nearness())
	}
	s.Tick(BaseTime(BandEasy) / 2)
	if s.Nearness() < 0.49 || s.Nearness() > 0.51 {
		t.Fatal(s.Nearness())
	}
}
