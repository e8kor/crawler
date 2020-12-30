package function

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
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

func Handle(r *http.Request) {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = os.Getenv("PG_USER")
		password = os.Getenv("PG_PASSWORD")
		dbname   = os.Getenv("PG_DBNAME")
		created  = time.Now()
		records  []Record
		payload  Entry
		result   sql.Result
		id       int64
	)

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		panic(err)
	}

	info := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", info)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for _, entry := range payload.Data {
		record := Record{
			Created: created,
			Data:    entry,
		}
		records = append(records, record)
	}

	result, err = db.Exec("INSERT INTO ? (created, data) VALUES ?", payload.Domain, records)
	if err != nil {
		panic(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		panic(err)
	}

	response := Result{
		Status:        true,
		Domain:        payload.Domain,
		IngestionTime: created,
		ID:            id,
	}

	w.Write([]byte(response))
}
