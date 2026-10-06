package engine

import (
	"math/rand/v2"
	"time"
)

const (
	// ProblemsPerLeg is how many correct answers reach the next gate.
	ProblemsPerLeg = 5

	// WrongPenalty is taken off the clock when a finished answer is wrong.
	// The same problem stays up, and the digits clear.
	WrongPenalty = 1500 * time.Millisecond

	// openingPace is the fresh time on the first leg. Every band shares it.
	// Easy, Medium, and Hard change the facts, not the speed.
	openingPace = 40 * time.Second

	// carryCap is the most saved time that comes along to the next gate.
	// The rest of a big lead turns into score, not a longer clock.
	carryCap = 4 * time.Second
)

// BaseTime is the opening clock. The band is ignored.
func BaseTime(_ Band) time.Duration {
	return openingPace
}

// paceDrop is how much fresh time the next gate loses once this many gates
// are already cleared. Six drops of 4s, then 3s, 3s, 2s, 2s, then 1s.
// The leg that starts gate 9 (8 cleared) is 10s. The leg that starts gate
// 12 (11 cleared) is 5s, about one second for each of the five problems.
func paceDrop(cleared int) time.Duration {
	switch {
	case cleared < 6:
		return 4 * time.Second
	case cleared < 8:
		return 3 * time.Second
	case cleared < 10:
		return 2 * time.Second
	default:
		return 1 * time.Second
	}
}

// LegPace is the fresh time for the leg after gates have already been
// cleared. Saved seconds are not included. It reaches zero and stays there.
func LegPace(gates int) time.Duration {
	if gates < 0 {
		gates = 0
	}
	pace := openingPace
	for i := 0; i < gates; i++ {
		pace -= paceDrop(i)
		if pace <= 0 {
			return 0
		}
	}
	return pace
}

// NextClock is the clock after gates have been cleared, with left still on
// the clock. Saved time carries, up to carryCap, on top of the fresh pace.
func NextClock(gates int, left time.Duration) time.Duration {
	if left < 0 {
		left = 0
	}
	if left > carryCap {
		left = carryCap
	}
	return LegPace(gates) + left
}

// Config selects the course. Band picks the facts. The clock does not follow it.
type Config struct {
	Op   Op
	Band Band
}

// Kind is what one digit, or one tick, did.
type Kind int

const (
	KindNone Kind = iota
	KindCorrect
	KindWrong
	KindGate
	KindEnded
)

// Result reports a digit or a tick. Points and Gate are set for KindGate.
// TimeLeft is the time still on the clock at the moment the gate was reached.
type Result struct {
	Kind     Kind
	Points   int
	Gate     int
	TimeLeft time.Duration
}

// Race is one run, from the countdown's GO until the clock hits zero.
type Race struct {
	cfg       Config
	src       Source
	problem   Problem
	typed     string
	legSolved int
	legWeight int
	timeLeft  time.Duration
	legClock  time.Duration
	score     int
	gates     int
	over      bool
}

// New starts a race on the first problem. src may be nil; then facts are random.
func New(cfg Config, src Source) *Race {
	if src == nil {
		src = NewGenerator(cfg.Op, cfg.Band, rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0x6d617468)))
	}
	budget := BaseTime(cfg.Band)
	return &Race{
		cfg:      cfg,
		src:      src,
		problem:  src.Next(),
		timeLeft: budget,
		legClock: budget,
	}
}

func (s *Race) Problem() Problem        { return s.problem }
func (s *Race) Typed() string           { return s.typed }
func (s *Race) TimeLeft() time.Duration { return s.timeLeft }
func (s *Race) LegClock() time.Duration { return s.legClock }
func (s *Race) Score() int              { return s.score }
func (s *Race) Gates() int              { return s.gates }
func (s *Race) LegSolved() int          { return s.legSolved }
func (s *Race) Over() bool              { return s.over }

// Nearness is 0 when the leg clock is full and 1 when it is empty.
// The window uses it to hurry the road when time is short. The gate's
// distance follows how many problems in the leg are already solved.
func (s *Race) Nearness() float64 {
	if s.legClock <= 0 {
		return 1
	}
	r := float64(s.timeLeft) / float64(s.legClock)
	if r < 0 {
		r = 0
	}
	if r > 1 {
		r = 1
	}
	return 1 - r
}

// TypeDigit appends d (0–9). The answer locks on the last box.
// A match shows the next problem immediately. A miss clears the boxes,
// keeps the same problem, and spends WrongPenalty.
func (s *Race) TypeDigit(d int) Result {
	if s.over || d < 0 || d > 9 {
		return Result{}
	}
	want := s.problem.AnswerDigits()
	if len(s.typed) >= len(want) {
		return Result{}
	}
	s.typed += string(rune('0' + d))
	if len(s.typed) < len(want) {
		return Result{}
	}
	if s.typed == want {
		return s.solve()
	}
	s.typed = ""
	s.timeLeft -= WrongPenalty
	if s.timeLeft > 0 {
		return Result{Kind: KindWrong}
	}
	s.timeLeft = 0
	s.over = true
	return Result{Kind: KindEnded}
}

// Backspace drops the last digit of an unfinished answer.
func (s *Race) Backspace() {
	if s.over || s.typed == "" {
		return
	}
	s.typed = s.typed[:len(s.typed)-1]
}

// Tick spends dt. The race ends when the clock reaches zero between gates.
func (s *Race) Tick(dt time.Duration) Result {
	if s.over || dt <= 0 {
		return Result{}
	}
	s.timeLeft -= dt
	if s.timeLeft > 0 {
		return Result{}
	}
	s.timeLeft = 0
	s.over = true
	return Result{Kind: KindEnded}
}

func (s *Race) solve() Result {
	s.typed = ""
	s.legSolved++
	s.legWeight += s.problem.Weight()
	if s.legSolved < ProblemsPerLeg {
		s.problem = s.src.Next()
		return Result{Kind: KindCorrect}
	}
	left := s.timeLeft
	// 10 points per difficulty-weight per second left. A photo finish still scores.
	points := s.legWeight * int(left.Milliseconds()) / 100
	if points < 1 {
		points = 1
	}
	s.score += points
	s.gates++
	next := NextClock(s.gates, left)
	s.timeLeft = next
	s.legClock = next
	s.legSolved = 0
	s.legWeight = 0
	s.problem = s.src.Next()
	return Result{Kind: KindGate, Points: points, Gate: s.gates, TimeLeft: left}
}
