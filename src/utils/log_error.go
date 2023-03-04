package utils

import (
	"os"
	"fmt"
	"log"
	"time"
	"sync"

	"github.com/fatih/color"
)

var mut sync.Mutex
// Thread-safe logging function that logs to "cultured_downloader.log" in the current working directory
func LogError(err error, errorMsg string, exit bool) {
	mut.Lock()
	defer mut.Unlock()

	if err == nil && errorMsg == "" {
		return
	}

	// log the error
	if err != nil {
		log.Println(color.RedString(err.Error()))
	}

	// write to log file
	f, fileErr := os.OpenFile(
		"cultured_downloader.log", 
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 
		0666,
	)
	if fileErr != nil {
		panic(fileErr)
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
		if err != nil {
			panic(err)
		} else {
			panic(errorMsg)
		}
	}
}