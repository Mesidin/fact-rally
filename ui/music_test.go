package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

func TestMusicTracksDecode(t *testing.T) {
	for _, blob := range [][]byte{cutTheCheck, fadeTrack} {
		if len(blob) < 100000 {
			t.Fatalf("track is %d bytes", len(blob))
		}
		stream, err := mp3.DecodeWithSampleRate(musicRate, bytes.NewReader(blob))
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 4096)
		if _, err := stream.Read(buf); err != nil {
			t.Fatal(err)
		}
		if pcmDuration(stream) < 30*time.Second {
			t.Fatalf("track is only %s", pcmDuration(stream))
		}
	}
}
