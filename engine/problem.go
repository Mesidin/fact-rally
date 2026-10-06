package engine

import (
	"fmt"
	"math/rand/v2"
	"strconv"
)

// Op is a course. Mixed picks a real operation for each problem.
type Op int

const (
	OpAdd Op = iota
	OpSub
	OpMul
	OpDiv
	OpMixed
)

// Band is how hard the facts in a course are.
type Band int

const (
	BandEasy Band = iota
	BandMedium
	BandHard
)

// Problem is one fact. Op is never Mixed.
type Problem struct {
	Left, Right int
	Op          Op
	Answer      int
}

func (o Op) Name() string {
	switch o {
	case OpAdd:
		return "Addition"
	case OpSub:
		return "Subtraction"
	case OpMul:
		return "Multiplication"
	case OpDiv:
		return "Division"
	case OpMixed:
		return "Mixed"
	default:
		return "Math"
	}
}

func (o Op) Key() string {
	switch o {
	case OpAdd:
		return "add"
	case OpSub:
		return "sub"
	case OpMul:
		return "mul"
	case OpDiv:
		return "div"
	case OpMixed:
		return "mixed"
	default:
		return "math"
	}
}

func (o Op) Symbol() string {
	switch o {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMul:
		return "×"
	case OpDiv:
		return "÷"
	default:
		return "?"
	}
}

func (b Band) Name() string {
	switch b {
	case BandEasy:
		return "Easy"
	case BandMedium:
		return "Medium"
	case BandHard:
		return "Hard"
	default:
		return "Easy"
	}
}

func (b Band) Key() string {
	switch b {
	case BandEasy:
		return "easy"
	case BandMedium:
		return "medium"
	case BandHard:
		return "hard"
	default:
		return "easy"
	}
}

// CourseKey identifies a best-score slot, for example "mul:hard".
func CourseKey(op Op, band Band) string {
	return op.Key() + ":" + band.Key()
}

func Courses() []Op {
	return []Op{OpAdd, OpSub, OpMul, OpDiv, OpMixed}
}

func Bands() []Band {
	return []Band{BandEasy, BandMedium, BandHard}
}

// Prompt is the equation without the answer, such as "8 × 7 =".
func (p Problem) Prompt() string {
	return fmt.Sprintf("%d %s %d =", p.Left, p.Op.Symbol(), p.Right)
}

// AnswerDigits is the exact character sequence the racer must type.
func (p Problem) AnswerDigits() string {
	return strconv.Itoa(p.Answer)
}

// Weight is the difficulty of one fact.
// 1 is a single-digit fact, 2 reaches into two digits, 3 is a long fact.
func (p Problem) Weight() int {
	if p.Answer >= 100 || (p.Left >= 20 && p.Right >= 10) || (p.Right >= 20 && p.Left >= 10) {
		return 3
	}
	if p.Answer >= 10 || p.Left >= 10 || p.Right >= 10 {
		return 2
	}
	return 1
}

// Source deals the next fact. The race asks for one as soon as the previous is solved.
type Source interface {
	Next() Problem
}

// Generator builds facts for one course and band.
type Generator struct {
	op      Op
	band    Band
	r       *rand.Rand
	prev    Problem
	hasPrev bool
}

func NewGenerator(op Op, band Band, r *rand.Rand) *Generator {
	if r == nil {
		r = rand.New(rand.NewPCG(1, 2))
	}
	return &Generator{op: op, band: band, r: r}
}

func (g *Generator) Next() Problem {
	p := g.roll()
	for i := 0; i < 16 && g.hasPrev && sameFact(p, g.prev); i++ {
		p = g.roll()
	}
	g.prev = p
	g.hasPrev = true
	return p
}

func sameFact(a, b Problem) bool {
	return a.Op == b.Op && a.Left == b.Left && a.Right == b.Right
}

func (g *Generator) roll() Problem {
	op := g.op
	if op == OpMixed {
		op = Op(g.r.IntN(4))
	}
	switch op {
	case OpSub:
		return g.sub()
	case OpMul:
		return g.mul()
	case OpDiv:
		return g.div()
	default:
		return g.add()
	}
}

func (g *Generator) between(lo, hi int) int {
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo + g.r.IntN(hi-lo+1)
}

func (g *Generator) add() Problem {
	var a, b int
	switch g.band {
	case BandMedium:
		sum := g.between(11, 20)
		a = g.between(0, sum)
		b = sum - a
	case BandHard:
		a = g.between(10, 60)
		b = g.between(10, 60)
		if a < 20 && b < 20 {
			b = g.between(20, 60)
		}
	default:
		a = g.between(0, 9)
		b = g.between(0, 10-a)
	}
	return Problem{Left: a, Right: b, Op: OpAdd, Answer: a + b}
}

func (g *Generator) sub() Problem {
	var a, b int
	switch g.band {
	case BandMedium:
		a = g.between(11, 20)
		b = g.between(0, a)
	case BandHard:
		a = g.between(30, 99)
		b = g.between(10, a)
	default:
		a = g.between(0, 10)
		b = g.between(0, a)
	}
	return Problem{Left: a, Right: b, Op: OpSub, Answer: a - b}
}

func (g *Generator) mul() Problem {
	var a, b int
	switch g.band {
	case BandMedium:
		a, b = g.pairAtLeast(0, 10, 6)
	case BandHard:
		if g.r.IntN(2) == 0 {
			hi := g.between(11, 12)
			lo := g.between(0, 12)
			a, b = hi, lo
		} else {
			a = g.between(10, 25)
			b = g.between(2, 9)
		}
		if g.r.IntN(2) == 0 {
			a, b = b, a
		}
	default:
		a = g.between(0, 5)
		b = g.between(0, 5)
	}
	return Problem{Left: a, Right: b, Op: OpMul, Answer: a * b}
}

func (g *Generator) div() Problem {
	switch g.band {
	case BandMedium:
		a, b := g.pairAtLeast(0, 10, 6)
		return g.divide(a, b)
	case BandHard:
		if g.r.IntN(2) == 0 {
			return g.divide(g.between(11, 12), g.between(1, 12))
		}
		quot := g.between(10, 25)
		divisor := g.between(2, 9)
		return Problem{Left: quot * divisor, Right: divisor, Op: OpDiv, Answer: quot}
	default:
		return g.divide(g.between(0, 5), g.between(0, 5))
	}
}

// pairAtLeast returns two factors in [lo, hi] with at least one >= floor.
func (g *Generator) pairAtLeast(lo, hi, floor int) (int, int) {
	for i := 0; i < 24; i++ {
		a := g.between(lo, hi)
		b := g.between(lo, hi)
		if a >= floor || b >= floor {
			return a, b
		}
	}
	return floor, g.between(lo, hi)
}

// divide builds an exact division fact. The divisor is never zero.
func (g *Generator) divide(a, b int) Problem {
	if a == 0 && b == 0 {
		b = 1
	}
	if b == 0 {
		a, b = b, a
	}
	dividend := a * b
	divisor, quot := b, a
	if a != 0 && g.r.IntN(2) == 0 {
		divisor, quot = a, b
	}
	return Problem{Left: dividend, Right: divisor, Op: OpDiv, Answer: quot}
}
