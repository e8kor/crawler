package function

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gocolly/colly/v2"

	handler "github.com/openfaas/templates-sdk/go-http"
)

// Entry stores Otodom dashboard structure
type Entry struct {
	Title  string `json:"title"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Price  string `json:"price"`
	Area   string `json:"area"`
	Link   string `json:"link"`
}

func Handle(r handler.Request) (handler.Response, error) {
	var response handler.Response

	query, err := url.ParseQuery(r.QueryString)
	if err != nil {
		return response, err
	}

	var (
		urls           = query["url"]
		destenationURL = r.Header.Get("X-Callback-Url")
	)

	if urls == nil {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}

	if urls == nil {
		log.Fatalln("missing url parameter")
	}
	entries := collectEntriess(urls)

	raw, err := json.Marshal(entries)
	if err != nil {
		return response, err
	}

	if destenationURL == "" {
		response = handler.Response{
			Body:       raw,
			StatusCode: http.StatusOK,
		}
		return response, nil
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

func collectEntriess(urls []string) []Entry {
	var entries []Entry
	c := colly.NewCollector()
	c.OnHTML("article[id]", func(e *colly.HTMLElement) {
		entry := Entry{
			Title:  e.ChildText("div.offer-item-details > header > h3 > a > span > span"),
			Name:   e.ChildText("div.offer-item-details-bottom > ul > li"),
			Region: e.ChildText("div.offer-item-details > header > p"),
			Price:  e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-price-per-m"),
			Area:   e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-area"),
			Link:   e.ChildAttr("div.offer-item-details > header > h3 > a", "href"),
		}
		entries = append(entries, entry)
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	for _, url := range urls {
		c.Visit(url)
	}
	return entries
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
