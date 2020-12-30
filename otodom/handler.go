package function

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
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

	var (
		SourceURL = r.URL.Query().Get("url")
		entries   = make([]Entry, 0, 20)
		err       error
		response  handler.Response
	)

	if SourceURL == "" {
		SourceURL = os.Getenv("SOURCE_URL")
	}
	if SourceURL == "" {
		log.Fatalln("{ \"error\": \"missing url parameter\"}")
		response = handler.Response{
			Body:       []byte("[]"),
			StatusCode: http.StatusOK,
		}
		return response, err
	}
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

	c.Visit(SourceURL)

	raw, err := json.Marshal(entries)
	if err != nil {
		return response, err
	}

	DestenationURL := r.Header.Get("X-Callback-Url")
	if DestenationURL == "" {
		response := handler.Response{
			Body:       raw,
			StatusCode: http.StatusOK,
		}
		return response, err
	}

	response, err = http.Post(DestenationURL, "application/json", bytes.NewBuffer(raw))
	if err != nil {
		return response, err
	}
	return response, err
}
