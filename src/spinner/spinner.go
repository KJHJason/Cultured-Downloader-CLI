package spinner

import (
	"os"
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const (
	CLEAR_LINE = "\033[K"

	// Common spinner types used in this program
	REQ_SPINNER  = "pong"
	JSON_SPINNER = "aesthetic"
	DL_SPINNER   = "material"
)

var (
	spinnerTypes map[string]SpinnerInfo
	colourMap  = map[string]color.Attribute{
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

func init() {
	spinnerTypes = GetSpinnerTypes()
	spinnersJson = nil // free up memory since it is no longer needed
}

// ListSpinnerTypes lists all the supported spinner types
func ListSpinnerTypes() {
	fmt.Println("Spinner types:")
	for spinnerType := range spinnerTypes {
		fmt.Printf(
			"%s\n",
			spinnerType,
		)
	}
} 

// ListColours lists all the supported colours
func ListColours() {
	fmt.Println("Colours:")
	for colour := range colourMap {
		fmt.Printf(
			"%s\n",
			colour,
		)
	}
}

// GetSpinner returns the spinner info of 
// the given spinner type string if it exists.
//
// If the spinner type string does not exist, the program will panic.
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
	ErrMsg     string

	count    int
	maxCount int
	active   bool
	mu       *sync.RWMutex
	stop     chan struct{}
}

// New creates a new spinner with the given spinner type, 
// colour, message, success message, error message and max count.
//
// For the spinner type and colour, please refer to the source code or 
// use ListSpinnerTypes() and ListColours() to print all the supported spinner types and colours.
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
		ErrMsg:     errMsg,

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
			for _, frame := range s.Spinner.Frames {
				select {
				case <-s.stop:
					return
				default:
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

// Add adds i to the spinner count
func (s *Spinner) Add(i int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count >= s.maxCount {
		return s.count
	}

	s.count += i
	return s.count
}

// UpdateMsg changes the spinner message
func (s *Spinner) UpdateMsg(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Msg = msg
}

// MsgIncrement increments the spinner count and 
// updates the message with the new count based onthe baseMsg.
//
// baseMsg should be a string with a single %d placeholder
// e.g. s.MsgIncrement("Downloading %d files...")
func (s* Spinner) MsgIncrement(baseMsg string) {
	s.UpdateMsg(
		fmt.Sprintf(
			baseMsg,
			s.Add(1),
		),
	)
}

func (s *Spinner) stopSpinner() {
	s.active = false
	if s.count != 0 {
		s.count = 0
	}
	s.stop <- struct{}{}
	close(s.stop)
}

// Stop stops the spinner and prints an outcome message
func (s *Spinner) Stop(hasErr bool) {
	s.StopWithFn(func () {
		if hasErr && s.ErrMsg != "" {
			color.Red(
				"\r✗ %s%s\n",
				s.ErrMsg,
				CLEAR_LINE,
			)
		} else if s.SuccessMsg != "" {
			color.Green(
				"\r✓ %s%s", 
				s.SuccessMsg,
				CLEAR_LINE,
			)
		}
	})
}

// Stop spinner with the given action function that will be called
func (s *Spinner) StopWithFn(action func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}

	s.stopSpinner()
	action()
}

// KillProgram stops the spinner, 
// prints the given message and exits the program with code 2.
//
// Used for Ctrl + C interrupts.
func (s *Spinner) KillProgram(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		os.Exit(2)
	}

	s.stopSpinner()
	color.Red(
		"\r✗ %s%s\n",
		msg,
		CLEAR_LINE,
	)
	os.Exit(2)
}
