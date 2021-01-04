package commons

import "time"

// Result is result of data storing
type Result struct {
	Status        bool      `json:"status"`
	Domain        string    `json:"domain"`
	IngestionTime time.Time `json:"ingestion_time"`
	Message       string    `json:"message"`
}
