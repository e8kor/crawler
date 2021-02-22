package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	framework "github.com/e8kor/crawler/commons"
	otodom "github.com/e8kor/crawler/otodom/commons"

	"github.com/gocolly/colly/v2"
)

//Handle is main function entrypoint
func Handle(w http.ResponseWriter, r *http.Request) {
	query, err := neturl.ParseQuery(r.URL.RawQuery)
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
			framework.HandleFailure(w, err)
			return
		}
	}
	framework.HandleSuccess(w, "saga completed")
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
				URL:  neturl.PathEscape(e.Attr("href")),
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
		wg           sync.WaitGroup
		raw          []byte
		httpResponse *http.Response
		schemas      = make(map[otodom.SchemaKey]otodom.Schema)
		entries      = make(map[otodom.SchemaKey][]otodom.Entry)
		created      = time.Now()
	)
	ch := make(chan otodom.CrawlingResponse, 40)
	wg.Add(len(pages))
	log.Printf("scheduling %d tasks", len(pages))
	for _, page := range pages {
		go getEntries(ch, gatewayPrefix, page)
	}

	go func() {
		for entry := range ch {
			key := otodom.SchemaKey{entry.SchemaName, entry.SchemaVersion}
			values, found := entries[key]
			if found {
				values = append(values, entry.Entries...)
			} else {
				values = entry.Entries
			}
			entries[key] = values
			schemas[key] = entry.Schema
			log.Printf("added %d for %s datasets, total count is %d \n", len(entry.Entries), key, len(values))
			wg.Done()
		}
	}()
	wg.Wait()
	close(ch)
	for key, value := range schemas {

		raw, err = preparePayload(created, key, value)

		log.Println("sending database persist request")
		httpResponse, err = http.Post(fmt.Sprintf("%s/database", gatewayPrefix), "application/json", bytes.NewBuffer(raw))
		if err != nil {
			log.Println("error when seding database persist request", err)
			return
		}
		log.Printf("received database response persist payload: %v\n", httpResponse)
	}

	for key, value := range entries {

		raw, err = preparePayload(created, key, value)
		if err != nil {
			log.Println("error when seding storage persist request", err)
			return
		}
		log.Println("sending storage persist request")
		httpResponse, err = http.Post(fmt.Sprintf("%s/storage", gatewayPrefix), "application/json", bytes.NewBuffer(raw))
		if err != nil {
			log.Println("error when seding storage persist request", err)
			return
		}
		log.Printf("received storage response persist payload: %v\n", httpResponse)
	}
	return
}

func preparePayload(created time.Time, key otodom.SchemaKey, schema interface{}) (bytes []byte, err error) {
	bytes, err = json.Marshal(schema)
	if err != nil {

		log.Println("error while marshalling Data", err)
		return
	}

	payload := framework.Entry{
		Created:       created,
		Domain:        "otodom",
		SchemaName:    key.SchemaName,
		SchemaVersion: key.SchemaVersion,
		Data:          []json.RawMessage{json.RawMessage(bytes)},
	}
	bytes, err = json.Marshal(payload)
	if err != nil {
		log.Println("error while marshalling Entry", err)
		return
	}
	return
}

func getEntries(ch chan otodom.CrawlingResponse, gatewayPrefix string, page otodom.Page) {

	var data otodom.CrawlingResponse

	log.Printf("sending otodom crawler request for %s\n", page.URL)
	response, err := http.Get(fmt.Sprintf("%s/otodom-crawler?url=%s", gatewayPrefix, page.URL))
	if err != nil {
		log.Println("failed to get response from scrapper", err)
		ch <- data
		return
	}

	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		log.Println("failed to read response from scrapper", err)
		ch <- data
		return
	}

	log.Printf("received response from crawler for %s\n", page.URL)

	ch <- data
	return
}
