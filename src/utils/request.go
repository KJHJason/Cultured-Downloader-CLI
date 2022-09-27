package utils

import (
	"io"
	"os"
	"fmt"
	"sync"
	"time"
	"strings"
	"net/url"
	"net/http"
)

func CallRequest(url string, timeout int, cookies []http.Cookie, method string, additionalHeaders map[string]string) (*http.Response, error) {
	// sends a request to the website
	req, err := http.NewRequest(method, url, nil)
	if (err != nil) {
		return nil, err
	}

	// add cookies to the request
	for _, cookie := range cookies {
		req.AddCookie(&cookie)
	}

	// add headers to the request
	for key, value := range additionalHeaders {
		req.Header.Add(key, value)
	}
	req.Header.Add(
		"User-Agent", USER_AGENT,
	)

	// send the request
	client := &http.Client{}
	client.Timeout = time.Duration(timeout) * time.Second
	resp, err := client.Do(req)
	if (err != nil) {
		fmt.Println(err)
		return nil, err
	}
	return resp, nil
}

func DownloadURL(fileURL string, filepath string, cookies []http.Cookie) error {
	var filename string
	var err error
	var resp *http.Response
	for i := 1; i <= RETRY_COUNTER; i++ {
		resp, err = CallRequest(fileURL, 30, cookies, "GET", map[string]string{})
		if (resp.StatusCode == 200 && err == nil) {
			filename, _ = url.PathUnescape(resp.Request.URL.String())
			break
		}
		if (i == RETRY_COUNTER) {
			// log the error
			errorMessage := fmt.Sprintf("failed to download %s after %d retries", fileURL, RETRY_COUNTER)
			LogError(err, errorMessage, false)
			return err
		}
	}
	defer resp.Body.Close()

	// check if the filename contains a query string
	if (strings.Contains(filename, "?")) {
		filename = filename[:strings.Index(filename, "?")]
	}

	filepath = filepath + filename[strings.LastIndex(filename, "/"):]
	// check if the file already exists
	if (PathExists(filepath)) {
		file, err := os.Open(filepath)
		if (err == nil) {
			defer file.Close()
			fileInfo, _ := file.Stat()
			if (fileInfo.Size() > 0) {
				return nil
			}
		}
	}

	// create the file
	out, err := os.Create(filepath)
	if (err != nil) {
		return err
	}
	defer out.Close()

	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if (err != nil) {
		os.Remove(filepath)
		return err
	}
	return nil
}

func DownloadURLsParallel(urls []map[string]string, cookies []http.Cookie) error {
	// downloads files from the urls in parallel
	// https://stackoverflow.com/a/25324090/16377492
	var wg sync.WaitGroup

	maxConcurrency := MAX_CONCURRENT_DOWNLOADS
	if (len(urls) < MAX_CONCURRENT_DOWNLOADS) {
		maxConcurrency = len(urls)
	}
	sem := make(chan struct{}, maxConcurrency)
	for _, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(url string, filepath string) {
			defer wg.Done()
			_ = DownloadURL(url, filepath, cookies)
			<-sem
		}(url["url"], url["filepath"])
	}
	close(sem) // close the channel to tell the goroutines that there are no more urls to download
	wg.Wait() // wait for all downloads to finish
	return nil
}