package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/e8kor/waader/log"
	handler "github.com/openfaas/templates-sdk/go-http"
)

// Entry is domain associated crawled json
type Entry struct {
	Domain string            `json:"domain"`
	Data   []json.RawMessage `json:"data"`
}

func Handle(r handler.Request) (handler.Response, error) {
	gatewayPrefix := os.Getenv("GATEWAY_URL")
	query, err := url.ParseQuery(r.QueryString)
	if err != nil {
		panic(err)
	}

	var (
		response handler.Response
		urls     = query["url"]
	)

	if urls == nil {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}
	crawlerResponse, err := http.Get(gatewayPrefix + "/otodom?url=" + strings.Join(urls[:], "&url="))
	if err != nil {
		panic(err)
	}

	persistorPayload := fmt.Sprintf(`{
		"domain": "otodom"
		"data": %s
	}`, string(streamToByte(crawlerResponse.Body)))

	log.Infoln("sending persist payload: " + persistorPayload)

	persistorResponse, err := http.Post(gatewayPrefix+"/persistor", "application/json", bytes.NewBuffer([]byte(persistorPayload)))
	if err != nil {
		panic(err)
	}
	response = handler.Response{
		Body:       streamToByte(persistorResponse.Body),
		StatusCode: persistorResponse.StatusCode,
		Header:     persistorResponse.Header,
	}
	return response, nil
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
