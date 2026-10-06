package engine

import (
	"math/rand/v2"
	"testing"
)

func TestPromptWeightAndDigits(t *testing.T) {
	p := Problem{Left: 8, Right: 7, Op: OpMul, Answer: 56}
	if p.Prompt() != "8 × 7 =" {
		t.Fatalf("prompt %q", p.Prompt())
	}
	if p.AnswerDigits() != "56" {
		t.Fatalf("digits %q", p.AnswerDigits())
	}
	if p.Weight() != 2 {
		t.Fatalf("weight %d", p.Weight())
	}
	if (Problem{Left: 3, Right: 4, Op: OpAdd, Answer: 7}).Weight() != 1 {
		t.Fatal("easy weight")
	}
	if (Problem{Left: 25, Right: 8, Op: OpMul, Answer: 200}).Weight() != 3 {
		t.Fatal("hard weight")
	}
	if CourseKey(OpDiv, BandHard) != "div:hard" {
		t.Fatal(CourseKey(OpDiv, BandHard))
	}
}

func TestGeneratorInvariants(t *testing.T) {
	bands := []Band{BandEasy, BandMedium, BandHard}
	ops := []Op{OpAdd, OpSub, OpMul, OpDiv, OpMixed}
	for _, op := range ops {
		for _, band := range bands {
			g := NewGenerator(op, band, rand.New(rand.NewPCG(uint64(op+1), uint64(band+3))))
			var prev Problem
			for i := 0; i < 300; i++ {
				p := g.Next()
				if err := checkProblem(op, band, p); err != nil {
					t.Fatalf("%s %s #%d: %v (%+v)", op.Name(), band.Name(), i, err, p)
				}
				if i > 0 && sameFact(p, prev) {
					t.Fatalf("%s %s repeated %+v", op.Name(), band.Name(), p)
				}
				prev = p
			}
		}
	}
}

func TestHardShapesBothAppear(t *testing.T) {
	g := NewGenerator(OpMul, BandHard, rand.New(rand.NewPCG(9, 9)))
	var teens, twoDigit int
	for i := 0; i < 200; i++ {
		p := g.Next()
		if teenTable(p.Left, p.Right) {
			teens++
		}
		if twoDigitFactor(p.Left, p.Right) {
			twoDigit++
		}
	}
	if teens == 0 || twoDigit == 0 {
		t.Fatalf("teens %d two-digit %d", teens, twoDigit)
	}

	d := NewGenerator(OpDiv, BandHard, rand.New(rand.NewPCG(4, 8)))
	var bigQuot, table int
	for i := 0; i < 200; i++ {
		p := d.Next()
		// Quotients past 12 only come from two-digit × one-digit.
		if p.Answer >= 13 {
			bigQuot++
		}
		// A divisor of 11 or 12 only comes from those tables.
		if p.Right >= 11 {
			table++
		}
	}
	if bigQuot == 0 || table == 0 {
		t.Fatalf("two-digit quotients %d table divisions %d", bigQuot, table)
	}
}

type problemError string

func (e problemError) Error() string { return string(e) }

func checkProblem(course Op, band Band, p Problem) error {
	if p.Op == OpMixed || p.Answer < 0 {
		return problemError("bad op or negative")
	}
	if course != OpMixed && p.Op != course {
		return problemError("wrong operation")
	}
	switch p.Op {
	case OpAdd:
		if p.Answer != p.Left+p.Right {
			return problemError("add")
		}
	case OpSub:
		if p.Right > p.Left || p.Answer != p.Left-p.Right {
			return problemError("sub")
		}
	case OpMul:
		if p.Answer != p.Left*p.Right {
			return problemError("mul")
		}
	case OpDiv:
		if p.Right == 0 || p.Left != p.Right*p.Answer {
			return problemError("div")
		}
	default:
		return problemError("op")
	}
	if !inBand(p, band) {
		return problemError("band")
	}
	return nil
}

func inBand(p Problem, band Band) bool {
	switch p.Op {
	case OpAdd:
		switch band {
		case BandEasy:
			return p.Left >= 0 && p.Left <= 9 && p.Right >= 0 && p.Answer <= 10
		case BandMedium:
			return p.Answer >= 11 && p.Answer <= 20 && p.Left >= 0 && p.Right >= 0
		default:
			return p.Left >= 10 && p.Left <= 60 && p.Right >= 10 && p.Right <= 60 && (p.Left >= 20 || p.Right >= 20)
		}
	case OpSub:
		switch band {
		case BandEasy:
			return p.Left >= 0 && p.Left <= 10 && p.Right >= 0
		case BandMedium:
			return p.Left >= 11 && p.Left <= 20 && p.Right >= 0
		default:
			return p.Left >= 30 && p.Left <= 99 && p.Right >= 10 && p.Right <= p.Left
		}
	case OpMul:
		switch band {
		case BandEasy:
			return inRange(p.Left, 0, 5) && inRange(p.Right, 0, 5)
		case BandMedium:
			return inRange(p.Left, 0, 10) && inRange(p.Right, 0, 10) && (p.Left >= 6 || p.Right >= 6)
		default:
			return teenTable(p.Left, p.Right) || twoDigitFactor(p.Left, p.Right)
		}
	case OpDiv:
		switch band {
		case BandEasy:
			return p.Right >= 1 && p.Right <= 5 && p.Answer >= 0 && p.Answer <= 5
		case BandMedium:
			return p.Right >= 1 && p.Right <= 10 && p.Answer >= 0 && p.Answer <= 10 && (p.Right >= 6 || p.Answer >= 6)
		default:
			table := (p.Right >= 11 && p.Right <= 12 && p.Answer >= 1 && p.Answer <= 12) ||
				(p.Answer >= 11 && p.Answer <= 12 && p.Right >= 1 && p.Right <= 12)
			two := p.Answer >= 10 && p.Answer <= 25 && p.Right >= 2 && p.Right <= 9
			return table || two
		}
	default:
		return false
	}
}

func inRange(n, lo, hi int) bool { return n >= lo && n <= hi }

func teenTable(a, b int) bool {
	return (a >= 11 && a <= 12 && b >= 0 && b <= 12) || (b >= 11 && b <= 12 && a >= 0 && a <= 12)
}

func twoDigitFactor(a, b int) bool {
	return (a >= 10 && a <= 25 && b >= 2 && b <= 9) || (b >= 10 && b <= 25 && a >= 2 && a <= 9)
}
