package function

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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
		created  = time.Now()
		records  []Record
		result   Result
		response handler.Response
		payload  Entry
		db       *sql.DB
		status   sql.Result
		err      error
		id       int64
	)

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return response, err
	}

	db, err = getDB()
	if err != nil {
		return response, err
	}
	defer db.Close()

	for _, entry := range payload.Data {
		record := Record{
			Created: created,
			Data:    entry,
		}
		records = append(records, record)
	}

	status, err = db.Exec("INSERT INTO ? (created, data) VALUES ?", payload.Domain, records)
	if err != nil {
		return response, err
	}

	id, err = status.LastInsertId()
	if err != nil {
		return response, err
	}

	result = Result{
		Status:        true,
		Domain:        payload.Domain,
		IngestionTime: created,
		ID:            id,
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return response, err
	}

	DestenationURL := r.Header.Get("X-Callback-Url")
	if DestenationURL == "" {
		response = handler.Response{
			Body:       raw,
			StatusCode: http.StatusOK,
		}
		return response, err
	}

	response, err = http.Post(DestenationURL, "application/json", bytes.NewBuffer(raw))
	if err != nil {
		return response, err
	}
	return response, err
}

func getDB() (*sql.DB, error) {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = os.Getenv("PG_USER")
		password = os.Getenv("PG_PASSWORD")
		dbname   = os.Getenv("PG_DBNAME")
	)
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	return sql.Open("postgres", connectionString)
}
