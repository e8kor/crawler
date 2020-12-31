package function

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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
	Message       string    `json:"message"`
}

// Handle a serverless request
func Handle(r handler.Request) (handler.Response, error) {
	var (
		destenationURL = r.Header.Get("X-Callback-Url")
		created        = time.Now()
		response       handler.Response
		result         Result
		payload        Entry
	)

	err := json.Unmarshal(r.Body, &payload)
	if err != nil {
		panic(err)
	}

	err = insertRecords(created, payload)
	if err != nil {
		message := fmt.Sprintf("error: %s", err)
		result = Result{
			Status:        false,
			Domain:        payload.Domain,
			IngestionTime: created,
			Message:       message,
		}
	} else {
		result = Result{
			Status:        false,
			Domain:        payload.Domain,
			IngestionTime: created,
		}
	}
	raw, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	if destenationURL == "" {
		response = handler.Response{
			Body:       raw,
			StatusCode: http.StatusOK,
		}
		return response, err
	}

	log.Println("using callback %s", destenationURL)
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

func insertRecords(created time.Time, entry Entry) error {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = getAPISecret("database-username")
		password = getAPISecret("database-password")
		dbname   = getAPISecret("database-name")
	)
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	var inserts []string
	for _, entry := range entry.Data {
		time := created.Format(time.RFC3339)
		j, err := entry.MarshalJSON()
		if err != nil {
			panic(err)
		}
		inserts = append(inserts, fmt.Sprintf("('%s'::timestamp, '%s')", time, j))
	}
	if inserts == nil {
		log.Println("no records to insert")
		return nil
	}
	insertStatement := strings.Join(inserts[:], ", ") + ";"
	statement := fmt.Sprintf("INSERT INTO %s(created, data) VALUES %s", entry.Domain, insertStatement)
	fmt.Println("statement is: ", statement)
	_, err = db.Exec(statement)
	if err != nil {
		return err
	}
	return nil
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func getAPISecret(secretName string) (secret string) {
	secretBytes, err := ioutil.ReadFile("/var/openfaas/secrets/" + secretName)
	if err != nil {
		panic(err)
	}
	secret = string(secretBytes)
	return secret
}
