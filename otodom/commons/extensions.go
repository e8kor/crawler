package commons

import (
	"regexp"
	"strings"
)

//ExtractNumber pattempt to parse number from string
func ExtractNumber(raw string) (number string) {
	pattern := regexp.MustCompile(`(\d+)`)
	items := pattern.FindAllStringSubmatch(raw, -1)
	for _, item := range items {
		number = number + item[1]
	}
	return number
}

//TakeChractersBefore take string before character
func TakeChractersBefore(raw string, predicate string) (result string) {
	return raw[:strings.Index(raw, predicate)]
}

//RetryAttempts generic to retry  collection
func RetryAttempts(retryCount int, action func() (data []interface{}, err error)) (data []interface{}) {
	for {
		result, err := action()
		if err != nil && retryCount > 0 {
			retryCount = retryCount - 1
			break
		}
		data = result
	}
	return
}
