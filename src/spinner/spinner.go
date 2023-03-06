package spinner

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	spinnerTypes = GetSpinnerTypes()
	colourMap = map[string]color.Attribute{
		"black":   color.FgBlack,
		"red":     color.FgRed,
		"green":   color.FgGreen,
		"yellow":  color.FgYellow,
		"blue":    color.FgBlue,
		"magenta": color.FgMagenta,
		"cyan":    color.FgCyan,
		"white":   color.FgWhite,
	}
)

func GetSpinner(spinnerType string) SpinnerInfo {
	if spinner, ok := spinnerTypes[spinnerType]; ok {
		spinner.Interval *= 0.001 
		return spinner
	} else {
		panic(
			fmt.Errorf(
				"error %d: spinner type %s not found",
				utils.DEV_ERROR,
				spinnerType,
			),
		)
	}
}

type Spinner struct {
	Spinner SpinnerInfo

	Colour     *color.Color
	Msg        string
	SuccessMsg string
	ErrMSg     string

	active bool
	mu     sync.Mutex
	stop   chan struct{}
}

func New(spinnerType, colour, message string) *Spinner {
	colourAttribute, ok := colourMap[colour]
	if !ok {
		panic(
			fmt.Errorf(
				"error %d: colour %s not found",
				utils.DEV_ERROR,
				colour,
			),
		)
	}

	return &Spinner{
		Spinner: GetSpinner(spinnerType),
		Colour: color.New(colourAttribute),
		Msg: message,
	}
}

// Starts the spinner
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return
	}

	s.active = true
	s.stop = make(chan struct{})

	go func() {
		for {
			select {
			case <-s.stop:
				return
			default:
				for _, frame := range s.Spinner.Frames {
					if !s.active {
						s.mu.Unlock()
						return
					}

					s.Colour.Printf("\r%s %s", frame, s.Msg)
					time.Sleep(time.Duration(s.Spinner.Interval) * time.Second)
				}
			}
		}
	}()
}

// Stop stops the spinner and prints an outcome message
func (s *Spinner) Stop(hasErr bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.stop <- struct{}{}
	close(s.stop)

	if hasErr {
		color.Red("\r✗ %s\n", s.ErrMSg)
	} else {
		color.Green("\r✓ %s\n", s.SuccessMsg)
	}
}
