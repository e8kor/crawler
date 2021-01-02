package function

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

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

// Page stores Otodom dashboard structure
type Page struct {
	URL  string
	Page int64
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
func findLastPage(url string) Page {
	var (
		lastPage Page
	)
	c := colly.NewCollector()
	c.OnHTML("#pagerForm > ul > li > a", func(e *colly.HTMLElement) {
		i, err := strconv.ParseInt(e.Text, 10, 64)
		if err != nil {
			log.Fatalln("error parsing last page", err)
		} else {
			page := Page{
				Page: i,
				URL:  e.Attr("href"),
			}
			if lastPage.Page < page.Page {
				lastPage = page
			}
		}
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("searching for last page on ", r.URL.String())
	})

	c.Visit(url)

	return lastPage
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
