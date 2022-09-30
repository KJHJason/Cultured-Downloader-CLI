package utils

import (
	"strings"
)

func SplitArgs(args string) []string {
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