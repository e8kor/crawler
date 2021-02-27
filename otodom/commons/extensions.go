package commons

import (
	"log"
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
func RetryAttempts(retryCount int, action func() ([]interface{}, error)) []interface{} {
	var (
		data []interface{}
		err  error
	)
	for {
		data, err = action()
		if err != nil {
			retryCount = retryCount - 1
		} else {
			retryCount = 0
		}
		if retryCount == 0 {
			if err != nil {
				log.Panicln("stop retrying on error", err)
			}
			break
		}
		log.Println("error:", err)
		log.Println("retry", retryCount, "...")
	}
	return data
}
