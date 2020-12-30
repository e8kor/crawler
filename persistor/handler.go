package function

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	handler "github.com/openfaas/templates-sdk/go-http"
)

// Entry is domain associated crawled json
type Entry struct {
	Domain string            `json:"domain"`
	Data   []json.RawMessage `json:"data"`
}

// Record is enriched Entry with metadata
type Record struct {
	Created time.Time       `json:"created"`
	Data    json.RawMessage `json:"data"`
}

// Result is result of data storing
type Result struct {
	Status        bool      `json:"status"`
	Domain        string    `json:"domain"`
	IngestionTime time.Time `json:"ingestion-time"`
	ID            int64     `json:"id"`
}

// Handle a serverless request
func Handle(r handler.Request) (handler.Response, error) {
	var (
		destenationURL = r.Header.Get("X-Callback-Url")
		created        = time.Now()
		response       handler.Response
		payload        Entry
	)

	err := json.Unmarshal(r.Body, &payload)
	if err != nil {
		return response, err
	}

	id, err := insertRecords(created, payload)
	if err != nil {
		return response, err
	}

	result := Result{
		Status:        true,
		Domain:        payload.Domain,
		IngestionTime: created,
		ID:            id,
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return response, err
	}

	if destenationURL == "" {
		response = handler.Response{
			Body:       raw,
			StatusCode: http.StatusOK,
		}
		return response, err
	}

	destenationResponse, err := http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
	if err != nil {
		return response, err
	}

	response = handler.Response{
		Body:       streamToByte(destenationResponse.Body),
		StatusCode: destenationResponse.StatusCode,
		Header:     destenationResponse.Header,
	}
	return response, err
}

func insertRecords(created time.Time, entry Entry) (int64, error) {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = os.Getenv("PG_USER")
		password = os.Getenv("PG_PASSWORD")
		dbname   = os.Getenv("PG_DBNAME")
		records  []Record
	)
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	for _, entry := range entry.Data {
		record := Record{
			Created: created,
			Data:    entry,
		}
		records = append(records, record)
	}

	status, err := db.Exec("INSERT INTO ? (created, data) VALUES ?", entry.Domain, records)
	if err != nil {
		return 0, err
	}

	return status.LastInsertId()
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
