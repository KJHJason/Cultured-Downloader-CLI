package utils

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
)

var (
	//go:embed icon.png
	iconImg []byte
	iconPath = filepath.Join(APP_PATH, "icon.png")
)

const Title = "Cultured Downloader CLI"

func writeIcon() error {
	defer func() {
		if iconImg != nil {
			iconImg = nil
		}
	}()

	if PathExists(iconPath) {
		return nil
	}

	f, err := os.Create(iconPath)
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, bytes.NewReader(iconImg)); err != nil {
		return err
	}
	return nil
}

// Alert shows a notification on the user's system with the given title and message.
func Alert(title, message string) error {	
	if err := writeIcon(); err != nil {
		return fmt.Errorf(
			"error %d: unable to write notification icon => %v", 
			UNEXPECTED_ERROR,
			err,
		)
	}

	if err := beeep.Alert(title, message, iconPath); err != nil {
		return fmt.Errorf(
			"error %d: unable to show notification => %v", 
			UNEXPECTED_ERROR,
			err,
		)
	}

	return nil
}

// AlertWithoutErr is the same as Alert but 
// if an error occurs, it will log it instead of returning it.
func AlertWithoutErr(title, message string) {
	if err := Alert(title, message); err != nil {
		LogError(err, "", false, ERROR)
	}
}
