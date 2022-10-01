package utils

import (
	"os"
	"fmt"
	"log"
	"time"
)

func LogError(err error, errorMsg string, exit bool) {
	if err == nil {
		return
	}

	log.Println(err)
	// write to log file
	f, fileErr := os.OpenFile("cultured_downloader.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileErr != nil {
		fmt.Println(fileErr)
		os.Exit(1)
	}
	defer f.Close()

	// From https://www.golangprograms.com/get-current-date-and-time-in-various-format-in-golang.html
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "%v: %v\n", now, err)
	if errorMsg != "" {
		fmt.Fprintf(f, "Additional info: %v\n\n", errorMsg)
	}

	if exit {
		os.Exit(1)
	}
}