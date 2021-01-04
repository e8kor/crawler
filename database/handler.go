package function

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	framework "github.com/e8kor/crawler/commons"
	//Database import for function
	_ "github.com/lib/pq"
)

// Handle a serverless request
func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		destenationURL = r.Header.Get("X-Callback-Url")
		ingestionTime  = time.Now()
		entry          framework.Entry
	)

	err := json.NewDecoder(r.Body).Decode(&entry)
	if err != nil {
		framework.HandleFailure(w, err)
		return
	}

	err = insert(entry, prepareConnectionString())

	if destenationURL != "" {
		log.Printf("using callback %s\n", destenationURL)
		result := entry.PrepareResult(ingestionTime, err)
		raw, err := json.Marshal(result)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		destenationResponse, err := http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		log.Printf("received x-callback-url %s response: %v\n", destenationURL, destenationResponse)
	}

	framework.HandleSuccess(w, "saved to database")
	return
}

func insert(entry framework.Entry, connection string) (err error) {

	statement, err := entry.PrepareInsertStatement()
	if err != nil {
		return
	}
	db, err := sql.Open("postgres", connection)
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(statement)
	return
}

func prepareConnectionString() (connection string) {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = framework.GetAPISecret("database-username")
		password = framework.GetAPISecret("database-password")
		dbname   = framework.GetAPISecret("database-name")
	)
	connection = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	return
}
