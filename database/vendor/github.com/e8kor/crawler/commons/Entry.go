package commons

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// Entry is domain associated crawled json
type Entry struct {
	Created time.Time         `json:"created"`
	Domain  string            `json:"domain"`
	Data    []json.RawMessage `json:"data"`
}

//PrepareInsertStatement generates sql insert query
func (entry *Entry) PrepareInsertStatement() (statement string, err error) {
	var (
		inserts []string
		time    = entry.Created.Format(time.RFC3339)
		bytes   []byte
	)
	for _, entry := range entry.Data {
		bytes, err = entry.MarshalJSON()
		if err != nil {
			return
		}
		inserts = append(inserts, fmt.Sprintf("('%s'::timestamp, '%s')", time, bytes))
	}
	if inserts == nil {
		log.Println("no records to insert")
		return
	}
	insertStatement := strings.Join(inserts[:], ", ")
	statement = fmt.Sprintf("INSERT INTO %s(created, data) VALUES %s;", entry.Domain, insertStatement)
	return
}

//PrepareResult is converting entry to Result
func (entry *Entry) PrepareResult(ingestionTime time.Time, err error) (result Result) {
	if err != nil {
		message := fmt.Sprintf("error: %s", err)
		result = Result{
			Status:        false,
			Domain:        entry.Domain,
			IngestionTime: ingestionTime,
			Message:       message,
		}
	} else {
		result = Result{
			Status:        true,
			Domain:        entry.Domain,
			IngestionTime: ingestionTime,
		}
	}
	return
}
