package utils

import (
	"os"
)

func PathExists(filepath string) bool {
	// checks if a file or directory exists
	_, err := os.Stat(filepath)
	return err == nil
}

func CreateDirectory(filepath string) error {
	// creates a directory
	err := os.MkdirAll(filepath, 0755)
	if (err != nil) {
		return err
	}
	return nil
}