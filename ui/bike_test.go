package ui

import (
	"image"
	"image/png"
	"testing"
)

func TestRearBikesShareTheTire(t *testing.T) {
	names := []string{
		"assets/bike_ride.png",
		"assets/bike_lean.png",
		"assets/bike_wheelie.png",
		"assets/bike_park.png",
	}
	var bounds image.Rectangle
	for _, name := range names {
		img := loadBike(t, name)
		if bounds.Empty() {
			bounds = img.Bounds()
		} else if img.Bounds() != bounds {
			t.Fatalf("%s bounds %v, want %v", name, img.Bounds(), bounds)
		}
		b := img.Bounds()
		if img.NRGBAAt(b.Min.X, b.Min.Y).A != 0 || img.NRGBAAt(b.Max.X-1, b.Max.Y-1).A != 0 {
			t.Fatalf("%s corners are not clear", name)
		}
		hit := false
		for dx := -4; dx <= 4; dx++ {
			if img.NRGBAAt(bikeAnchorX+dx, bikeAnchorY).A > 180 {
				hit = true
			}
		}
		if !hit {
			t.Fatalf("%s rear tire does not meet the anchor", name)
		}
	}

	flame := loadBike(t, "assets/bike_wheelie.png")
	found := false
	b := flame.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !found; y += 3 {
		for x := b.Min.X; x < b.Max.X; x += 3 {
			c := flame.NRGBAAt(x, y)
			if c.A > 200 && c.R > 220 && c.G > 90 && c.G < 210 && c.B < 90 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("boost pose has no exhaust flame")
	}
}

func loadBike(t *testing.T, name string) *image.NRGBA {
	t.Helper()
	f, err := bikeFiles.Open(name)
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
