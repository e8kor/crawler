package commons

import (
	"encoding/json"
	"log"
	"time"

	framework "github.com/e8kor/crawler/commons"
)

// Field stores Otodom schema field
type Field struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Schema stores Otodom schema
type Schema struct {
	Title      Field `json:"title"`
	Name       Field `json:"name"`
	Region     Field `json:"region"`
	Price      Field `json:"price"`
	TotalPrice Field `json:"total_price"`
	Area       Field `json:"area"`
	Link       Field `json:"link"`
}

// CrawlingResponse stores Otodom schema
type CrawlingResponse struct {
	SchemaName    string        `json:"schema_name"`
	SchemaVersion string        `json:"schema_version"`
	Schema        interface{}   `json:"schema"`
	Entries       []interface{} `json:"entries"`
}

// SchemaKey store key for schema
type SchemaKey struct {
	SchemaName    string `json:"schema_name"`
	SchemaVersion string `json:"schema_version"`
}

// NewEntry create new record
func NewEntry(
	domain string,
	created time.Time,
	key SchemaKey,
	data interface{},
) (entry framework.Entry, err error) {
	raw, err := json.Marshal(data)
	if err != nil {
		log.Println("error while marshalling data for", err)
		return
	}
	entry = framework.Entry{
		Created:       created,
		Domain:        domain,
		SchemaName:    key.SchemaName,
		SchemaVersion: key.SchemaVersion,
		Data:          json.RawMessage(raw),
	}
	return
}
