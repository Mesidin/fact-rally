package main

import (
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"mathskill/internal/meta"
	"mathskill/profile"
	"mathskill/ui"
)

func main() {
	ebiten.SetWindowSize(ui.ScreenW, ui.ScreenH)
	ebiten.SetWindowTitle(meta.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	store, err := profile.Load()
	if err != nil {
		log.Printf("profiles: %v", err)
	}
	if err := ebiten.RunGame(ui.NewApp(store)); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
}
