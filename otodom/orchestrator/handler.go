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
	"strconv"
	"strings"
	"sync"
	"time"

	otodom "github.com/e8kor/crawler/otodom/commons"

	"github.com/gocolly/colly/v2"
	handler "github.com/openfaas/templates-sdk/go-http"
)

//Handle is main function entrypoint
func Handle(r handler.Request) (response handler.Response, err error) {
	query, err := url.ParseQuery(r.QueryString)
	if err != nil {
		return
	}

	var (
		urls          = query["url"]
		gatewayPrefix = os.Getenv("GATEWAY_URL")
	)

	if urls == nil {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}
	for _, url := range urls {
		pages := collectPages(url)
		err = processPages(gatewayPrefix, pages)
		if err != nil {
			return
		}
	}

	response = handler.Response{
		Body:       []byte(`{ "message": "saga completed"}`),
		StatusCode: http.StatusOK,
		Header:     r.Header,
	}
	return
}

func collectPages(url string) (pages []otodom.Page) {
	var (
		lastPage otodom.Page
		c        = colly.NewCollector()
	)

	c.OnHTML("#pagerForm > ul > li > a", func(e *colly.HTMLElement) {
		i, err := strconv.Atoi(e.Text)
		if err != nil {
			log.Println("error parsing page", err)
		} else {
			page := otodom.Page{
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

	log.Printf("found last page %v\n", lastPage)

	for i := 1; i < lastPage.Page; i++ {
		var pageURL string
		if strings.Contains(url, "?") {
			pageURL = fmt.Sprintf("%s&page=%d", url, i)
		} else {
			pageURL = fmt.Sprintf("%s?page=%d", url, i)
		}
		pages = append(pages, otodom.Page{
			Page: i,
			URL:  pageURL,
		})
	}

	log.Printf("found %d pages\n", len(pages))

	return
}

func processPages(gatewayPrefix string, pages []otodom.Page) (err error) {
	var (
		responses    []json.RawMessage
		wg           sync.WaitGroup
		raw          []byte
		httpResponse *http.Response
	)
	ch := make(chan []json.RawMessage, 40)
	wg.Add(len(pages))
	log.Printf("scheduling %d tasks", len(pages))
	for _, page := range pages {
		go getEntries(ch, gatewayPrefix, page)
	}

	go func() {
		for json := range ch {
			responses = append(responses, json...)
			log.Printf("added %d datasets, total count is %d \n", len(json), len(responses))
			wg.Done()
		}
	}()
	wg.Wait()
	close(ch)

	log.Printf("collected %d datasets\n", len(responses))

	raw, err = json.Marshal(Entry{
		Created: time.Now(),
		Domain:  "otodom",
		Data:    responses,
	})
	if err != nil {
		log.Println("error while marshalling Entry", err)
		return
	}

	log.Println("sending database persist request")
	httpResponse, err = http.Post(fmt.Sprintf("%s/database", gatewayPrefix), "application/json", bytes.NewBuffer(raw))
	if err != nil {
		log.Println("error when seding database persist request", err)
		return
	}
	log.Printf("received database response persist payload: %v\n", httpResponse)

	log.Println("sending storage persist request")
	httpResponse, err = http.Post(fmt.Sprintf("%s/storage", gatewayPrefix), "application/json", bytes.NewBuffer(raw))
	if err != nil {
		log.Println("error when seding storage persist request", err)
		return
	}
	log.Printf("received storage response persist payload: %v\n", httpResponse)
	return
}

func getEntries(ch chan []json.RawMessage, gatewayPrefix string, page otodom.Page) {

	var data []json.RawMessage

	log.Printf("sending otodom crawler request for %s\n", page.URL)
	response, err := http.Get(fmt.Sprintf("%s/otodom-scrapper?url=%s", gatewayPrefix, page.URL))
	if err != nil {
		log.Println("failed to get response from scrapper", err)
		ch <- data
		return
	}

	err = json.Unmarshal(streamToByte(response.Body), &data)
	if err != nil {
		log.Println("failed to read response from scrapper", err)
		ch <- data
		return
	}

	log.Printf("received response from crawler for %s\n", page.URL)

	ch <- data
	return
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
