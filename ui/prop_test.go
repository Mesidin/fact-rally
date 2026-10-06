package ui

import (
	"image"
	"image/png"
	"testing"
)

func TestPropsAreNightSprites(t *testing.T) {
	sizes := []struct {
		name string
		w, h int
	}{
		{"assets/prop_tree.png", 36, 58},
		{"assets/prop_lamp.png", 24, 64},
		{"assets/prop_board.png", 52, 40},
		{"assets/prop_plus.png", 16, 30},
		{"assets/prop_post.png", 8, 52},
		{"assets/prop_bar.png", 8, 6},
	}
	for _, tc := range sizes {
		img := loadProp(t, tc.name)
		b := img.Bounds()
		if b.Dx() != tc.w || b.Dy() != tc.h {
			t.Fatalf("%s bounds %v, want %dx%d", tc.name, b, tc.w, tc.h)
		}
	}
	for _, name := range []string{"assets/prop_tree.png", "assets/prop_lamp.png", "assets/prop_board.png", "assets/prop_plus.png"} {
		img := loadProp(t, name)
		if img.NRGBAAt(0, 0).A != 0 {
			t.Fatalf("%s top-left is not clear", name)
		}
	}

	tree := loadProp(t, "assets/prop_tree.png")
	if tree.NRGBAAt(treeAnchorX, treeAnchorY).A < 180 {
		t.Fatal("palm trunk does not meet the ground")
	}
	if !hasRim(tree, true) || !hasRim(tree, false) {
		t.Fatal("palm is missing a cyan left edge or a magenta right edge")
	}
	lamp := loadProp(t, "assets/prop_lamp.png")
	if lamp.NRGBAAt(lampAnchorX, lampAnchorY).A < 180 {
		t.Fatal("lamp pole does not meet the ground")
	}
	if !hasBulb(lamp) {
		t.Fatal("lamp has no bulb")
	}
	plus := loadProp(t, "assets/prop_plus.png")
	if plus.NRGBAAt(plusAnchorX, plusAnchorY).A < 180 {
		t.Fatal("plus sign does not meet the ground")
	}
	if !hasWhite(plus) {
		t.Fatal("plus sign has no light pixel to tint")
	}
	post := loadProp(t, "assets/prop_post.png")
	if post.NRGBAAt(postAnchorX, postAnchorY).A < 180 {
		t.Fatal("gate post does not meet the ground")
	}
}

func hasRim(img *image.NRGBA, left bool) bool {
	b := img.Bounds()
	mid := b.Min.X + b.Dx()/2
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if left && x >= mid || !left && x < mid {
				continue
			}
			c := img.NRGBAAt(x, y)
			if c.A < 200 {
				continue
			}
			if left && c.R < 120 && c.G > 180 && c.B > 180 {
				return true
			}
			if !left && c.R > 200 && c.G < 90 && c.B > 140 {
				return true
			}
		}
	}
	return false
}

func hasBulb(img *image.NRGBA) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A > 200 && c.R > 220 && c.G > 180 && c.B > 140 && c.B < 230 {
				return true
			}
		}
	}
	return false
}

func hasWhite(img *image.NRGBA) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A > 200 && c.R > 220 && c.G > 220 && c.B > 220 {
				return true
			}
		}
	}
	return false
}

func loadProp(t *testing.T, name string) *image.NRGBA {
	t.Helper()
	f, err := propFiles.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	return out
}
