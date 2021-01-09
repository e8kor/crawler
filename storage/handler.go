package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	framework "github.com/e8kor/crawler/commons"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var client *minio.Client

// Handle a serverless request
func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		destenationURL = r.Header.Get("X-Callback-Url")
		ingestionTime  = time.Now()
		raw            []byte
		payload        framework.Entry
		httpResponse   *http.Response
	)

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		framework.HandleFailure(w, err)
		return
	}

	err = insert(payload)
	if err != nil {
		framework.HandleFailure(w, err)
		return
	}

	if destenationURL != "" {
		log.Printf("using callback %s\n", destenationURL)
		raw, err = json.Marshal(framework.Result{
			Status:        true,
			Domain:        payload.Domain,
			IngestionTime: ingestionTime,
		})
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		httpResponse, err = http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		log.Printf("received x-callback-url %s response: %v\n", destenationURL, httpResponse)
	}

	framework.HandleSuccess(w, "saved to storage")
	return
}

func insert(entry framework.Entry) (err error) {
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
	filename, err := framework.RandomFilename()
	if err != nil {
		log.Println("failed generating filename", err)
		return
	}
	path := fmt.Sprintf("schema_name=%s/schema_version=%s/created=%d/%s.json", entry.SchemaName, entry.SchemaVersion, entry.Created.Unix(), filename)
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
		accessKeyID     = framework.GetAPISecret("storage-access-key")
		secretAccessKey = framework.GetAPISecret("storage-secret-key")
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
