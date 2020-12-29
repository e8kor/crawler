package function

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Entry is domain associated crawled json
type Entry struct {
	domain string
	data   []map[string]interface{}
}

// Handle a serverless request

func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = os.Getenv("PG_USER")
		password = os.Getenv("PG_PASSWORD")
		dbname   = os.Getenv("PG_DBNAME")
		payload  Entry
		result   sql.Result
		id       int64
	)

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		panic(err)
	}

	info := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", info)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	preparedQuery := fmt.Sprintf(`INSERT INTO %s (data) VALUES (%s)`, payload.domain, payload.data)
	result, err = db.Exec(preparedQuery)
	if err != nil {
		panic(err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		panic(err)
	}

	response := fmt.Sprintf(`{
		"status": true,
		"id": "%s"
	}`, id)

	w.Write([]byte(response))
}
