package textparser

import (
	"bufio"
	"os"
	"io"
	"fmt"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const PAGE_NUM_REGEX_GRP_NAME = "pageNum"

var PAGE_NUM_REGEX_STR = fmt.Sprintf(
	`(?:; (?P<%s>[1-9]\d*(?:-[1-9]\d*)?))?`,
	PAGE_NUM_REGEX_GRP_NAME,
)

// openTextFile opens the text file at the given path and returns a os.File and a bufio.Reader.
//
// If an error occurs, the program will exit with an error message and status code 1.
func openTextFile(textFilePath, website string) (*os.File, *bufio.Reader) {
	f, err := os.Open(textFilePath)
	if err != nil {
		errMsg := fmt.Sprintf(
			"error %d: failed to open %s cookie file at %s, more info => %v",
			utils.OS_ERROR,
			website,
			textFilePath,
			err,
		)
		color.Red(errMsg)
		os.Exit(1)
	}
	return f, bufio.NewReader(f)
}

// readLine reads a line from the given reader and returns the line as a slice of bytes.
//
// If the reader reaches EOF, the second return value will be true. Otherwise, it will be false.
// However, if an error occurs, the program will exit with an error message and status code 1.
func readLine(reader *bufio.Reader, textFilePath, website string) ([]byte, bool) {
	lineBytes, err := utils.ReadLine(reader)
	if err != nil {
		if err == io.EOF {
			return nil, true
		}
		errMsg := fmt.Sprintf(
			"error %d: failed to read %s text file at %s, more info => %v",
			utils.OS_ERROR,
			website,
			textFilePath,
			err,
		)
		color.Red(errMsg)
		os.Exit(1)
	}
	return lineBytes, false
}
