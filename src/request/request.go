package request

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func CallRequest(method, url string, timeout int, cookies []http.Cookie, additionalHeaders, params map[string]string, checkStatus bool) (*http.Response, error) {
	// sends a request to the website
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// add cookies to the request
	for _, cookie := range cookies {
		if strings.Contains(url, cookie.Domain) {
			req.AddCookie(&cookie)
		}
	}

	// add headers to the request
	for key, value := range additionalHeaders {
		req.Header.Add(key, value)
	}
	req.Header.Add(
		"User-Agent", utils.USER_AGENT,
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
	for i := 1; i <= utils.RETRY_COUNTER; i++ {
		resp, err := client.Do(req)
		if err == nil {
			if !checkStatus {
				return resp, nil
			} else if resp.StatusCode == 200 {
				return resp, nil
			}
		}
		time.Sleep(utils.GetRandomDelay())
	}
	errorMessage := fmt.Sprintf("failed to send a request to %s after %d retries", url, utils.RETRY_COUNTER)
	utils.LogError(err, errorMessage, false)
	return nil, err
}

func DownloadURL(fileURL, filePath string, cookies []http.Cookie, headers, params map[string]string) error {
	downloadTimeout := 25 * 60  // 25 minutes in seconds as downloads 
								// can take quite a while for large files (especially for Pixiv)
								// However, the average max file size on these platforms is around 300MB.
								// Note: Fantia do have a max file size per post of 3GB if one paid extra for it.
	resp, err := CallRequest("GET", fileURL, downloadTimeout, cookies, headers, params, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check if filepath already have a filename attached
	if filepath.Ext(filePath) == "" {
		os.MkdirAll(filePath, 0755)
		filename, err := url.PathUnescape(resp.Request.URL.String())
		if err != nil {
			panic(err)
		}
		filename = utils.GetLastPartOfURL(filename)
		filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
		filePath = filepath.Join(filePath, filenameWithoutExt + strings.ToLower(filepath.Ext(filename)))
	} else {
		filePathDir := filepath.Dir(filePath)
		os.MkdirAll(filePathDir, 0755)
		filePathWithoutExt := strings.TrimSuffix(filePath, filepath.Ext(filePath))
		filePath = filepath.Join(filePathDir, filePathWithoutExt + strings.ToLower(filepath.Ext(filePath)))
	}

	// check if the file already exists
	if empty, _ := utils.CheckIfFileIsEmpty(filePath); !empty {
		return nil
	}

	// create the file
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}

	// write the body to file
	// https://stackoverflow.com/a/11693049/16377492
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		file.Close()
		os.Remove(filePath)
		errorMsg := fmt.Sprintf("failed to download %s due to %v", fileURL, err)
		utils.LogError(err, errorMsg, false)
		return nil
	}
	file.Close()
	return nil
}

func DownloadURLsParallel(urls []map[string]string, maxConcurrency int, cookies []http.Cookie, headers, params map[string]string) {
	if len(urls) < maxConcurrency {
		maxConcurrency = len(urls)
	}

	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	for _, url := range urls {
		wg.Add(1)
		queue <- struct{}{}
		go func(fileUrl, filePath string) {
			defer wg.Done()
			DownloadURL(fileUrl, filePath, cookies, headers, params)
			<-queue
		}(url["url"], url["filepath"])
	}
	close(queue)
	wg.Wait()
}