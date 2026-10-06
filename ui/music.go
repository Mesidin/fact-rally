package ui

import (
	"bytes"
	_ "embed"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// NCS releases, played in this order and then repeated.
// Credit lives in the README. M quiets the music.
// The motorcycle engine in sound.go is kept and not started.
//
//go:embed assets/music/cut-the-check.mp3
var cutTheCheck []byte

//go:embed assets/music/fade.mp3
var fadeTrack []byte

const (
	musicRate = 44100
	// musicVolume is half of the earlier level. The engine is silent.
	musicVolume = 0.2
)

type song struct {
	player *audio.Player
	dur    time.Duration
}

// playlist is the background music and the motorcycle engine.
// A nil playlist is silent.
type playlist struct {
	songs   []song
	i       int
	muted   bool
	engine  *engineVoice
	engineP *audio.Player
}

func newPlaylist() *playlist {
	ctx := audio.NewContext(musicRate)
	p := &playlist{}
	// startEngine(ctx) stays available in sound.go. The whine was grating,
	// so the player is left nil until a better one is ready.
	blobs := [][]byte{cutTheCheck, fadeTrack}
	for _, blob := range blobs {
		stream, err := mp3.DecodeWithSampleRate(musicRate, bytes.NewReader(blob))
		if err != nil {
			log.Printf("music: %v", err)
			continue
		}
		player, err := ctx.NewPlayer(stream)
		if err != nil {
			log.Printf("music: %v", err)
			continue
		}
		player.SetVolume(musicVolume)
		p.songs = append(p.songs, song{player: player, dur: pcmDuration(stream)})
	}
	if len(p.songs) > 0 {
		p.songs[0].player.Play()
	}
	return p
}

// hear aims the engine. A nil engine stays quiet. on is false on the menus,
// so a running whine fades out.
func (p *playlist) hear(speed, boost, scrape float64, on bool) {
	if p == nil || p.engine == nil {
		return
	}
	p.engine.set(speed, boost, scrape, on)
}

func pcmDuration(stream *mp3.Stream) time.Duration {
	if stream.SampleRate() <= 0 || stream.Length() <= 0 {
		return 0
	}
	seconds := float64(stream.Length()) / float64(stream.SampleRate()*4)
	return time.Duration(seconds * float64(time.Second))
}

// update keeps the playlist moving. allowToggle is false on the name
// screen, where M is a letter in the racer's name.
func (p *playlist) update(allowToggle bool) {
	if p == nil {
		return
	}
	if allowToggle && inpututil.IsKeyJustPressed(ebiten.KeyM) {
		p.muted = !p.muted
		if p.muted {
			p.pauseHeard()
		} else {
			p.resumeHeard()
		}
		return
	}
	if p.muted || len(p.songs) == 0 {
		return
	}
	if p.songs[p.i].done() {
		p.advance()
	}
}

func (s song) done() bool {
	if s.player.IsPlaying() || s.dur <= 0 {
		return false
	}
	return s.player.Position()+300*time.Millisecond >= s.dur
}

func (p *playlist) pauseHeard() {
	if len(p.songs) > 0 {
		p.songs[p.i].player.Pause()
	}
	if p.engineP != nil {
		p.engineP.Pause()
	}
}

func (p *playlist) resumeHeard() {
	if len(p.songs) > 0 {
		p.songs[p.i].player.Play()
	}
	if p.engineP != nil {
		p.engineP.Play()
	}
}

func (p *playlist) advance() {
	prev := p.songs[p.i].player
	prev.Pause()
	if err := prev.Rewind(); err != nil {
		log.Printf("music: %v", err)
	}
	p.i = (p.i + 1) % len(p.songs)
	p.songs[p.i].player.Play()
}
