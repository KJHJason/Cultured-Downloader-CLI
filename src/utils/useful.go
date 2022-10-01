package utils

import (
	"strings"
)

func SplitArgs(args string) []string {
	if args == "" {
		return []string{}
	}

	splittedArgs := strings.Split(args, ",")
	seen := make(map[string]bool)
	arr := []string{}
	for _, el := range splittedArgs {
		if _, value := seen[el]; !value {
			seen[el] = true
			arr = append(arr, el)
		}
	}
	return arr
}

func GetLastPartOfURL(url string) string {
	if strings.Contains(url, "?") {
		url = url[:strings.Index(url, "?")]
	}
	splitted := strings.Split(url, "/")
	return splitted[len(splitted)-1]
}