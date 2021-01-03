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

	"github.com/gocolly/colly/v2"
	handler "github.com/openfaas/templates-sdk/go-http"
)

// Page stores Otodom dashboard structure
type Page struct {
	URL  string
	Page int
}
type PageSorter []Page

func (a PageSorter) Len() int           { return len(a) }
func (a PageSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PageSorter) Less(i, j int) bool { return a[i].Page < a[j].Page }

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
		results       []json.RawMessage
	)

	if urls == nil {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}
	wg := sync.WaitGroup{}
	for _, url := range urls {
		for _, page := range collectPages(url) {
			wg.Add(1)
			go func(page Page) {
				defer wg.Done()
				log.Printf("sending otodom crawler request for %s\n", page.URL)
				rawJSON, err := getEntries(gatewayPrefix, page)
				if err != nil {
					return
				}
				log.Printf("response from otodom crawler url %s\n%v\n", page.URL, rawJSON)
				results = append(results, rawJSON...)
			}(page)
		}
	}
	wg.Wait()

	raw, err := json.Marshal(Entry{
		Created: time.Now(),
		Domain:  "otodom",
		Data:    results,
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

func collectPages(url string) []Page {
	var (
		pages    []Page
		lastPage Page
		c        = colly.NewCollector()
	)

	c.OnHTML("#pagerForm > ul > li > a", func(e *colly.HTMLElement) {
		i, err := strconv.Atoi(e.Text)
		if err != nil {
			log.Println("error parsing page", err)
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

	log.Printf("found last page %v\n", lastPage)

	for i := 1; i < lastPage.Page; i++ {
		var pageURL string
		if strings.Contains(url, "?") {
			pageURL = fmt.Sprintf("%s&page=%d", url, i)
		} else {
			pageURL = fmt.Sprintf("%s?page=%d", url, i)
		}
		pages = append(pages, Page{
			Page: i,
			URL:  pageURL,
		})
	}

	log.Printf("found %d pages\n", len(pages))

	return pages
}

func getEntries(gatewayPrefix string, page Page) (rawJSON []json.RawMessage, err error) {
	response, err := http.Get(fmt.Sprintf("%s/otodom-scrapper?url=%s", gatewayPrefix, page.URL))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(streamToByte(response.Body), &rawJSON)
	if err != nil {
		return nil, err
	}
	return rawJSON, err
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
