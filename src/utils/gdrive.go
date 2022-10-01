package utils

import (
	"regexp"
	"net/http"
)

func GDriveKeyIsValid(api_key string) bool {
	match, _ := regexp.MatchString(`^AIza[\w-]{35}$`, api_key)
	if !match {
		return false
	}

	res, err := CallRequest(
		"https://www.googleapis.com/drive/v3/files", 
		5, 
		[]http.Cookie{}, 
		"GET",
		nil,
		map[string]string{"key": api_key},
	)
	if (err != nil) {
		panic(err)
	}
	defer res.Body.Close()
	return res.StatusCode != 400
}