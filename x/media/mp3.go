package media

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type MP3Player struct {
	done chan bool
	ctrl *beep.Ctrl
}

func NewMp3Player() AudioPlayer {
	return &MP3Player{
		done: make(chan bool),
		ctrl: &beep.Ctrl{Paused: false},
	}
}

func (p *MP3Player) Play(mp3Data []byte) error {
	streamer, format, err := mp3.Decode(io.NopCloser(bytes.NewReader(mp3Data)))
	if err != nil {
		return fmt.Errorf("failed to decode mp3: %w", err)
	}
	defer streamer.Close()

	// make it done at the end of streaming
	p.ctrl.Streamer = beep.Seq(streamer, beep.Callback(func() {
		p.done <- true
	}))

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	speaker.Play(p.ctrl)

	<-p.done
	return nil
}

func (p *MP3Player) Stop() error {
	p.ctrl.Paused = true
	select {
	case p.done <- true:
	default:
	}
	return nil
}
