package spinner

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const (
	CLEAR_LINE = "\033[K"

	// Standard spinner types used
	REQ_SPINNER  = "aesthetic"
	JSON_SPINNER = "pong"
	DL_SPINNER   = "material"
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

		"fgHiBlack":   color.FgHiBlack,
		"fgHiRed":     color.FgHiRed,
		"fgHiGreen":   color.FgHiGreen,
		"fgHiYellow":  color.FgHiYellow,
		"fgHiBlue":    color.FgHiBlue,
		"fgHiMagenta": color.FgHiMagenta,
		"fgHiCyan":    color.FgHiCyan,
		"fgHiWhite":   color.FgHiWhite,
	}
)

func GetSpinner(spinnerType string) SpinnerInfo {
	if spinner, ok := spinnerTypes[spinnerType]; ok {
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

// Thread-safe spinner
type Spinner struct {
	Spinner SpinnerInfo

	Colour     *color.Color
	Msg        string
	SuccessMsg string
	ErrMSg     string

	count    int
	maxCount int
	active   bool
	mu       *sync.RWMutex
	stop     chan struct{}
}

func New(spinnerType, colour, message, successMsg, errMsg string, maxCount int) *Spinner {
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

		Colour:     color.New(colourAttribute),
		Msg:        message,
		SuccessMsg: successMsg,
		ErrMSg:     errMsg,

		count:    0,
		maxCount: maxCount,
		active:   false,
		mu:       &sync.RWMutex{},
		stop:     make(chan struct{}, 1),
	}
}

// Starts the spinner
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}

	s.active = true
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-s.stop:
				return
			default:
				for _, frame := range s.Spinner.Frames {
					s.mu.Lock()
					if !s.active {
						s.mu.Unlock()
						return
					}

					s.Colour.Printf(
						"\r%s %s%s", 
						frame, 
						s.Msg, 
						CLEAR_LINE,
					)
					s.mu.Unlock()
					time.Sleep(
						time.Duration(s.Spinner.Interval) * time.Millisecond,
					)
				}
			}
		}
	}()
}

func (s *Spinner) Add(i int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count >= s.maxCount {
		return s.count
	}

	s.count += i
	return s.count
}

func (s *Spinner) UpdateMsg(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Msg = msg
}

func (s* Spinner) MsgIncrement(baseMsg string) {
	s.UpdateMsg(
		fmt.Sprintf(
			s.Msg,
			s.Add(1),
		),
	)
}

// Stop stops the spinner and prints an outcome message
func (s *Spinner) Stop(hasErr bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}

	s.active = false
	if s.count != 0 {
		s.count = 0
	}
	s.stop <- struct{}{}
	close(s.stop)

	if hasErr {
		color.Red(
			"\r✗ %s%s\n",
			s.ErrMSg,
			CLEAR_LINE,
		)
	} else {
		color.Green(
			"\r✓ %s%s", 
			s.SuccessMsg,
			CLEAR_LINE,
		)
	}
}
