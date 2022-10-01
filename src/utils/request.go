package utils

import (
	"io"
	"os"
	"fmt"
	"sync"
	"time"
	"net/url"
	"net/http"
	"path/filepath"
	"github.com/panjf2000/ants/v2"
)

func CallRequest(url string, timeout int, cookies []http.Cookie, 
	method string, additionalHeaders map[string]string, params map[string]string) (*http.Response, error) {
	// sends a request to the website
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// add cookies to the request
	for _, cookie := range cookies {
		req.Header.Add("Cookie", cookie.String())
	}

	// add headers to the request
	for key, value := range additionalHeaders {
		req.Header.Add(key, value)
	}
	req.Header.Add(
		"User-Agent", USER_AGENT,
	)

	// add params to the request
	if len(params) > 0 {
		query := req.URL.Query()
		for key, value := range params {
			query.Add(key, value)
		}
		req.URL.RawQuery = query.Encode()
	}

	// send the request
	client := &http.Client{}
	client.Timeout = time.Duration(timeout) * time.Second
	for i := 1; i <= RETRY_COUNTER; i++ {
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		} else {
			time.Sleep(time.Duration(RETRY_DELAY) * time.Second)
		}
	}
	errorMessage := fmt.Sprintf("failed to send a request to %s after %d retries", url, RETRY_COUNTER)
	LogError(err, errorMessage, false)
	return nil, err
}

func DownloadURL(fileURL string, filePath string, cookies []http.Cookie) error {
	var filename string
	resp, err := CallRequest(fileURL, 30, cookies, "GET", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check if the filename contains a query string
	filename, _ = url.PathUnescape(resp.Request.URL.String())
	filePath = filepath.Join(filePath, GetLastPartOfURL(filename))
	if empty, _ := CheckIfFileIsEmpty(filePath); !empty {
		return nil
	}

	// create the file
	out, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// write the body to file
	// https://stackoverflow.com/a/11693049/16377492
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(filePath)
		panic(err)
	}
	return nil
}

func DownloadURLsParallel(urls []map[string]string, cookies []http.Cookie) {
	maxConcurrency := MAX_CONCURRENT_DOWNLOADS
	if len(urls) < MAX_CONCURRENT_DOWNLOADS {
		maxConcurrency = len(urls)
	}
	pool, _ := ants.NewPool(maxConcurrency)
	defer pool.Release()

	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		fileURL := url["url"]
		filePath := url["path"]
		pool.Submit(func() {
			defer wg.Done()
			DownloadURL(fileURL, filePath, cookies)
		})
	}

	// wait for all the tasks to finish
	wg.Wait()
}