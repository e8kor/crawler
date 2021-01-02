package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	handler "github.com/openfaas/templates-sdk/go-http"
)

// Entry is domain associated crawled json
type Entry struct {
	Created time.Time         `json:"created"`
	Domain  string            `json:"domain"`
	Data    []json.RawMessage `json:"data"`
}

func Handle(r handler.Request) (handler.Response, error) {
	query, err := url.ParseQuery(r.QueryString)
	if err != nil {
		panic(err)
	}

	var (
		response      handler.Response
		urls          = query["url"]
		gatewayPrefix = os.Getenv("GATEWAY_URL")
		rawJSON       []json.RawMessage
	)

	if urls == nil {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}

	log.Println("sending otodom crawler request")
	queryArguments := fmt.Sprintf("?url=%s", strings.Join(urls[:], "&url="))
	crawlerResponse, err := http.Get(fmt.Sprintf("%s/otodom%s", gatewayPrefix, queryArguments))
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(streamToByte(crawlerResponse.Body), &rawJSON)
	if err != nil {
		return response, err
	}

	raw, err := json.Marshal(Entry{
		Created: time.Now(),
		Domain:  "otodom",
		Data:    rawJSON,
	})
	if err != nil {
		return response, err
	}

	log.Printf("sending persist payload: %s\n", string(raw))
	databaseResponse, err := http.Post(fmt.Sprintf("%s/database", gatewayPrefix), "application/json", bytes.NewBuffer(raw))
	if err != nil {
		return response, err
	}

	log.Printf("received database response persist payload: %v\n", databaseResponse)
	storageResponse, err := http.Post(fmt.Sprintf("%s/storage", gatewayPrefix), "application/json", bytes.NewBuffer(raw))
	if err != nil {
		return response, err
	}
	log.Printf("received storage response persist payload: %v\n", storageResponse)
	response = handler.Response{
		Body:       []byte("saga completed"),
		StatusCode: databaseResponse.StatusCode,
		Header:     databaseResponse.Header,
	}
	return response, nil
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
