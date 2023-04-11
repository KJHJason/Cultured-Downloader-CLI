// Spinners from https://github.com/sindresorhus/cli-spinners/blob/main/spinners.json
package spinner

import (
	_ "embed"
	"encoding/json"
)

type SpinnerInfo struct {
	Interval int64
	Frames   []string
}

//go:embed spinners.json
var spinnersJson []byte

// GetSpinnerTypes returns a map of spinnerInfo 
// that contains the interval and frames for each spinner.
func GetSpinnerTypes() map[string]SpinnerInfo {
	if spinnerTypes != nil {
		// Return the cached spinner types
		return spinnerTypes
	}

	var spinners map[string]SpinnerInfo
	if err := json.Unmarshal(spinnersJson, &spinners); err != nil {
		panic(err)
	}

	return spinners
}
