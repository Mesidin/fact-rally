package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"mathskill/engine"
	"mathskill/internal/meta"
)

func TestTitleFitsTheHomeScreen(t *testing.T) {
	w, _ := text.Measure(meta.Title, face(68), 0)
	if w > float64(ScreenW-80) {
		t.Fatalf("title %q is %v px wide", meta.Title, w)
	}
}

func TestOperatorGlyphs(t *testing.T) {
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	samples := "×÷+-=·0123456789"
	for _, op := range engine.Courses() {
		if op == engine.OpMixed {
			continue
		}
		samples += op.Symbol()
	}
	seen := map[rune]bool{}
	for _, r := range samples {
		if seen[r] {
			continue
		}
		seen[r] = true
		idx, err := f.GlyphIndex(&buf, r)
		if err != nil || idx == 0 {
			t.Errorf("missing glyph %q err %v", r, err)
		}
	}
}

func TestCommas(t *testing.T) {
	if commas(0) != "0" || commas(999) != "999" || commas(1000) != "1,000" || commas(12000) != "12,000" {
		t.Fatalf("%s %s %s %s", commas(0), commas(999), commas(1000), commas(12000))
	}
}
