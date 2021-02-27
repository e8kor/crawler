package function

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	nurl "net/url"
	"os"
	"sync"
	"time"

	framework "github.com/e8kor/crawler/commons"
	otodom "github.com/e8kor/crawler/otodom/commons"
)

//Handle is main function entrypoint
func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		empty         interface{}
		urls          []string
		paramURL      = r.URL.Query().Get("url")
		crawlerSuffix = os.Getenv("CRAWLER_SUFFIX")
		pagesSuffix   = os.Getenv("PAGES_SUFFIX")
		domain        = os.Getenv("DOMAIN")
		created       = time.Now()
		pages         []otodom.Page
	)

	if urls != nil {
		urls = append(urls, paramURL)
	} else {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}
	for _, url := range urls {
		params := nurl.Values{}
		params.Add("url", url)
		err := framework.CallFunction(pagesSuffix, params, empty, pages)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		schemas, entries := processPages(crawlerSuffix, pages)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}

		for key, value := range schemas {
			params := nurl.Values{}
			payload, err := otodom.NewEntry(domain, created, key, value)
			if err != nil {
				framework.HandleFailure(w, err)
				return
			}
			err = framework.FireFunction("/database", params, payload)
			if err != nil {
				framework.HandleFailure(w, err)
				return
			}
		}

		for key, value := range entries {
			payload, err := otodom.NewEntry(domain, created, key, value)
			if err != nil {
				framework.HandleFailure(w, err)
				return
			}
			err = framework.FireFunction("/storage", params, payload)
			if err != nil {
				framework.HandleFailure(w, err)
				return
			}
		}
		return
	}
	framework.HandleSuccess(w, "saga completed")
	return
}

func collectPages(gatewayPrefix string, pagesSuffix string, pageURL string) (pages []otodom.Page, err error) {
	var (
		params = url.Values{}
		data   []otodom.Page
	)
	log.Println("sending collect total pages request")
	params.Add("url", pageURL)
	response, err := http.Post(gatewayPrefix+pagesSuffix+"?"+params.Encode(), "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Println("error when sending collect total pages request", err)
		return nil, err
	}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		log.Println("failed to read collect total pages response", err)
		return nil, err
	}
	return data, nil
}

func processPages(
	crawlerSuffix string,
	pages []otodom.Page,
) (
	schemas map[otodom.SchemaKey]interface{},
	entries map[otodom.SchemaKey][]interface{},
) {
	var (
		wg sync.WaitGroup
	)
	ch := make(chan otodom.CrawlingResponse, 40)
	wg.Add(len(pages))
	log.Println("scheduling", len(pages), "tasks")
	for _, page := range pages {
		go getEntries(ch, crawlerSuffix, page)
	}

	go func() {
		for entry := range ch {
			if entry.SchemaName == "" || entry.SchemaVersion == "" {
				log.Printf("skipping entry %+v\n", entry)
				wg.Done()
			} else {
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
		}
	}()
	wg.Wait()
	close(ch)
	return
}

func getEntries(ch chan otodom.CrawlingResponse, crawlerSuffix string, page otodom.Page) {
	var (
		data  otodom.CrawlingResponse
		empty interface{}
	)

	log.Println("sending otodom crawler request for", page.URL)
	params := url.Values{}
	params.Add("url", page.URL)
	err := framework.CallFunction(crawlerSuffix, params, empty, data)
	if err != nil {
		log.Println("failed to get response from scrapper", err)
		ch <- data
		return
	}

	log.Println("received response from crawler for", page.URL)
	ch <- data
	return
}
