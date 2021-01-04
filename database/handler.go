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

func (entry *Entry) prepareInsertStatement() (statement string, err error) {
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

// Result is result of data storing
type Result struct {
	Status        bool      `json:"status"`
	Domain        string    `json:"domain"`
	IngestionTime time.Time `json:"ingestion_time"`
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

	if destenationURL != "" {
		log.Printf("using callback %s\n", destenationURL)
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
				Status:        true,
				Domain:        payload.Domain,
				IngestionTime: ingestionTime,
			}
		}
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

func insert(entry Entry) (err error) {
	var (
		host     = os.Getenv("PG_HOST")
		port     = os.Getenv("PG_PORT")
		user     = getAPISecret("database-username")
		password = getAPISecret("database-password")
		dbname   = getAPISecret("database-name")
	)
	statement, err := entry.prepareInsertStatement()
	if err != nil {
		return
	}
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(statement)
	return
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
