package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
)

func logJsonResponse(body []byte) {
	var prettyJson bytes.Buffer
	err := json.Indent(&prettyJson, body, "", "    ")
	if err != nil {
		color.Red(
			fmt.Sprintf(
				"error %d: failed to indent JSON response body due to %v",
				JSON_ERROR,
				err,
			),
		)
		return
	}

	filename := fmt.Sprintf("saved_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join("json", filename)
	os.MkdirAll(filepath.Dir(filePath), 0666)
	err = os.WriteFile(filePath, prettyJson.Bytes(), 0666)
	if err != nil {
		color.Red(
			fmt.Sprintf(
				"error %d: failed to write JSON response body to file due to %v",
				UNEXPECTED_ERROR,
				err,
			),
		)
	}
}

// Read the response body and unmarshal it into a interface and returns it
func LoadJsonFromResponse(res *http.Response, format any) error {
	body, err := ReadResBody(res)
	if err != nil {
		return err
	}

	// write to file if debug mode is on
	if DEBUG_MODE {
		logJsonResponse(body)
	}

	if err = json.Unmarshal(body, &format); err != nil {
		err = fmt.Errorf(
			"error %d: failed to unmarshal json response from %s due to %v\nBody: %s",
			RESPONSE_ERROR,
			res.Request.URL.String(),
			err,
			string(body),
		)
		return err
	}
	return nil
}

func LoadJsonFromBytes(body []byte, format any) error {
	if err := json.Unmarshal(body, &format); err != nil {
		err = fmt.Errorf(
			"error %d: failed to unmarshal json due to %v\nBody: %s",
			JSON_ERROR,
			err,
			string(body),
		)
		return err
	}
	return nil
}
