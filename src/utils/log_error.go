package utils

import (
	"os"
	"fmt"
	"log"
	"time"
	"sync"
)

func LogError(err error, errorMsg string, exit bool) {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	if err == nil && errorMsg == "" {
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
	if err != nil && errorMsg != "" {
		fmt.Fprintf(f, "%v: %v\n", now, err)
		if errorMsg != "" {
			fmt.Fprintf(f, "Additional info: %v\n\n", errorMsg)
		}
	} else if err != nil {
		fmt.Fprintf(f, "%v: %v\n\n", now, err)
	} else {
		fmt.Fprintf(f, "%v: %v\n\n", now, errorMsg)
	}

	if exit {
		os.Exit(1)
	}
}