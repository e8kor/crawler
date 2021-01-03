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

	//Database import for function
	_ "github.com/lib/pq"

	handler "github.com/openfaas/templates-sdk/go-http"
)

// Entry is domain associated crawled json
type Entry struct {
	Created time.Time         `json:"created"`
	Domain  string            `json:"domain"`
	Data    []json.RawMessage `json:"data"`
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
		ingestionTime  = time.Now()
		response       handler.Response
		result         Result
		payload        Entry
	)

	err := json.Unmarshal(r.Body, &payload)
	if err != nil {
		return response, err
	}

	err = insert(payload)
	if err != nil {
		message := fmt.Sprintf("error: %s", err)
		result = Result{
			Status:        false,
			Domain:        payload.Domain,
			IngestionTime: ingestionTime,
			Message:       message,
		}
	} else {
		result = Result{
			Status:        false,
			Domain:        payload.Domain,
			IngestionTime: ingestionTime,
		}
	}
	if destenationURL != "" {
		log.Printf("using callback %s\n", destenationURL)
		raw, err := json.Marshal(result)
		if err != nil {
			return response, err
		}
		destenationResponse, err := http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
		if err != nil {
			return response, err
		}
		log.Printf("received x-callback-url %s response: %v\n", destenationURL, destenationResponse)
	}

	response = handler.Response{
		Body:       []byte(`{ "message": "saved to database"}`),
		StatusCode: http.StatusOK,
		Header:     r.Header,
	}
	return response, err
}

func insert(entry Entry) error {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = getAPISecret("database-username")
		password = getAPISecret("database-password")
		dbname   = getAPISecret("database-name")
		inserts  []string
		time     = entry.Created.Format(time.RFC3339)
	)
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, entry := range entry.Data {
		j, err := entry.MarshalJSON()
		if err != nil {
			return err
		}
		inserts = append(inserts, fmt.Sprintf("('%s'::timestamp, '%s')", time, j))
	}
	if inserts == nil {
		log.Println("no records to insert")
		return nil
	}
	insertStatement := strings.Join(inserts[:], ", ")
	statement := fmt.Sprintf("INSERT INTO %s(created, data) VALUES %s;", entry.Domain, insertStatement)
	fmt.Printf("statement is: %s\n", statement)
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
