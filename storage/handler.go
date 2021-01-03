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
		Body:       []byte(`{ "message": "saved to storage"}`),
		StatusCode: http.StatusOK,
		Header:     r.Header,
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
		location        = "us-east-1"
	)
	client, err := minio.New(
		endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		},
	)
	exists, err := client.BucketExists(ctx, entry.Domain)
	if err == nil && exists {
		log.Printf("We already own %s\n", entry.Domain)
	} else if err != nil {
		log.Fatalln(err)
		return err
	} else if !exists {
		log.Printf("creating bucket %s\n", entry.Domain)
		err = client.MakeBucket(ctx, entry.Domain, minio.MakeBucketOptions{Region: location})
		if err != nil {
			log.Println("error creating bucket", err)
			return err
		}
		log.Printf("Successfully created %s\n", entry.Domain)
	}

	raw, err := json.Marshal(entry.Data)
	if err != nil {
		log.Println("failed marshalling data", err)
		return err
	}
	filename, err := randomFilename()
	if err != nil {
		log.Println("failed generating filename", err)
		return err
	}
	path := fmt.Sprintf("created=%d/%s.json", entry.Created.Unix(), filename)
	log.Printf("writing json at path %s", path)
	status, err := client.PutObject(
		ctx,
		entry.Domain,
		path,
		bytes.NewBuffer(raw),
		-1,
		minio.PutObjectOptions{
			ContentType: "application/json",
		},
	)
	if err != nil {
		log.Println("failed writing data", err)
		return err
	}
	log.Printf("upload status: %v\n", status)
	return nil
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
	return secret
}
