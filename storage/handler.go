package function

import (
	"bytes"
	"context"
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

	if destenationURL == "" {
		response = handler.Response{
			Body:       []byte(`{ "message": "saved to storage"}`),
			StatusCode: http.StatusOK,
		}
		return response, nil
	}

	log.Printf("using callback %s\n", destenationURL)
	raw, err := json.Marshal(result)
	if err != nil {
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
	return response, nil
}

func insert(entry Entry) error {
	var (
		ctx             = context.Background()
		endpoint        = os.Getenv("MINIO_HOST")
		accessKeyID     = getAPISecret("storage-access-key")
		secretAccessKey = getAPISecret("storage-secret-key")
		useSSL          = false
		location        = ""
	)
	client, err := minio.New(
		endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		},
	)
	err = client.MakeBucket(ctx, entry.Domain, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := client.BucketExists(ctx, entry.Domain)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", entry.Domain)
		} else {
			log.Fatalln(err)
			return err
		}
	} else {
		log.Printf("Successfully created %s\n", entry.Domain)
	}
	raw, err := json.Marshal(entry.Data)
	if err != nil {
		return err
	}
	status, err := client.PutObject(
		ctx,
		entry.Domain,
		fmt.Sprintf("/created=%d/%d.json", entry.Created.Unix(), entry.Created.Unix()),
		bytes.NewBuffer(raw),
		-1,
		minio.PutObjectOptions{
			ContentType: "application/json",
		},
	)
	if err != nil {
		return err
	}
	log.Printf("upload status: %v\n", status)
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
