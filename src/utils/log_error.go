package utils

import (
	"context"
	"os"
	"fmt"
	"log"
	"time"
	"path/filepath"

	"github.com/fatih/color"
)

const LogSuffix = "\n\n"
var (
	mainLogger *logger
	logFolder = filepath.Join(APP_PATH, "logs")
	logFilePath = filepath.Join(
		logFolder,
		fmt.Sprintf(
			"cultured_downloader-cli_v%s_%s.log", 
			VERSION, 
			time.Now().Format("2006-01-02"),
		),
	)
)

func init() {
	// will be opened througout the program's runtime
	// hence, there is no need to call f.Close() at the end of this function
	f, fileErr := os.OpenFile(
		logFilePath, 
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 
		0666,
	)
	if fileErr != nil {
		fileErr = fmt.Errorf(
			"error opening log file: %v\nlog file path: %s", 
			fileErr, 
			logFilePath,
		)
		log.Println(color.RedString(fileErr.Error()))
		os.Exit(1)
	}
	mainLogger = NewLogger(f)
}

// Delete all empty log files and log files
// older than 30 days except for the current day's log file.
func DeleteEmptyAndOldLogs() error {
	err := filepath.Walk(logFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || path == logFilePath {
			return nil
		}

		if info.Size() == 0 || info.ModTime().Before(time.Now().AddDate(0, 0, -30)) {
			return os.Remove(path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Thread-safe logging function that logs to "cultured_downloader.log" in the logs directory
func LogError(err error, errorMsg string, exit bool, level int) {
	if err == nil && errorMsg == "" {
		return
	}

	if err != nil && errorMsg != "" {
		mainLogger.LogBasedOnLvl(level, err.Error() + LogSuffix)
		if errorMsg != "" {
			mainLogger.LogBasedOnLvlf(level, "Additional info: %v%s", errorMsg, LogSuffix)
		}
	} else if err != nil {
		mainLogger.LogBasedOnLvl(level, err.Error() + LogSuffix)
	} else {
		mainLogger.LogBasedOnLvlf(level, errorMsg + LogSuffix)
	}

	if exit {
		if err != nil {
			color.Red(err.Error())
		} else {
			color.Red(errorMsg)
		}
		os.Exit(1)
	}
}

// Uses the thread-safe LogError() function to log a slice of errors or a channel of errors
//
// Also returns if any errors were due to context.Canceled which is caused by Ctrl + C.
func LogErrors(exit bool, errChan chan error, level int, errs ...error) bool {
	if errChan != nil && len(errs) > 0 {
		panic(
			fmt.Sprintf(
				"error %d: cannot pass both an error channel and a slice of errors to LogErrors()",
				DEV_ERROR,
			),
		)
	}

	hasCanceled := false
	if errChan != nil {
		for err := range errChan {
			if err == context.Canceled {
				if !hasCanceled {
					hasCanceled = true
				}
				continue
			}
			LogError(err, "", exit, level)
		}
		return hasCanceled
	}

	for _, err := range errs {
		if err == context.Canceled {
			if !hasCanceled {
				hasCanceled = true
			}
			continue
		}
		LogError(err, "", exit, level)
	}
	return hasCanceled
}
