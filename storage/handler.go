package function

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	IngestionTime time.Time `json:"ingestion_time"`
	Message       string    `json:"message"`
}

var client *minio.Client

// Handle a serverless request
func Handle(r handler.Request) (response handler.Response, err error) {
	var (
		destenationURL = r.Header.Get("X-Callback-Url")
		ingestionTime  = time.Now()
		raw            []byte
		payload        Entry
		httpResponse   *http.Response
	)

	err = json.Unmarshal(r.Body, &payload)
	if err != nil {
		return
	}

	err = insert(payload)
	if err != nil {
		return
	}

	if destenationURL != "" {
		log.Printf("using callback %s\n", destenationURL)
		raw, err = json.Marshal(Result{
			Status:        true,
			Domain:        payload.Domain,
			IngestionTime: ingestionTime,
		})
		if err != nil {
			return
		}
		httpResponse, err = http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
		if err != nil {
			return
		}
		log.Printf("received x-callback-url %s response: %v\n", destenationURL, httpResponse)
	}

	response = handler.Response{
		Body:       []byte(`{ "message": "saved to storage"}`),
		StatusCode: http.StatusOK,
		Header:     r.Header,
	}
	return
}

func insert(entry Entry) (err error) {
	var (
		ctx      = context.Background()
		location = "us-east-1"
	)
	if client == nil {
		client, err = createClient()
		if err != nil {
			return
		}
	}
	exists, err := client.BucketExists(ctx, entry.Domain)
	if err == nil && exists {
		log.Printf("We already own %s\n", entry.Domain)
	} else if err != nil {
		log.Fatalln(err)
		return
	} else if !exists {
		log.Printf("creating bucket %s\n", entry.Domain)
		err = client.MakeBucket(ctx, entry.Domain, minio.MakeBucketOptions{Region: location})
		if err != nil {
			log.Println("error creating bucket", err)
			return
		}
		log.Printf("Successfully created %s\n", entry.Domain)
	}

	raw, err := json.Marshal(entry.Data)
	if err != nil {
		log.Println("failed marshalling data", err)
		return
	}
	filename, err := randomFilename()
	if err != nil {
		log.Println("failed generating filename", err)
		return
	}
	path := fmt.Sprintf("created=%d/%s.json", entry.Created.Unix(), filename)
	log.Printf("writing json at path %s", path)
	buff := bytes.NewBuffer(raw)

	status, err := client.PutObject(
		ctx,
		entry.Domain,
		path,
		buff,
		-1,
		minio.PutObjectOptions{
			ContentType: "application/json",
		},
	)
	if err != nil {
		log.Println("failed writing data", err)
		return
	}
	log.Printf("upload status: %v\n", status)
	return
}

func createClient() (client *minio.Client, err error) {
	var (
		endpoint        = os.Getenv("MINIO_HOST")
		accessKeyID     = getAPISecret("storage-access-key")
		secretAccessKey = getAPISecret("storage-secret-key")
		useSSL          = false
	)
	return minio.New(
		endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		},
	)
}

func randomFilename() (s string, err error) {
	b := make([]byte, 8)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	s = fmt.Sprintf("%x", b)
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
	return
}
