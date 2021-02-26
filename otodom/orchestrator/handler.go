package function

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	framework "github.com/e8kor/crawler/commons"
	otodom "github.com/e8kor/crawler/otodom/commons"
)

//Handle is main function entrypoint
func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		urls          []string
		paramURL      = r.URL.Query().Get("url")
		gatewayPrefix = os.Getenv("GATEWAY_URL")
		crawlerSuffix = os.Getenv("CRAWLER_SUFFIX")
		pagesSuffix   = os.Getenv("PAGES_SUFFIX")
		domain        = os.Getenv("DOMAIN")
		created       = time.Now()
	)

	if urls != nil {
		urls = append(urls, paramURL)
	} else {
		urls = append(urls, os.Getenv("SOURCE_URL"))
	}
	for _, url := range urls {
		pages, err := collectPages(gatewayPrefix, crawlerSuffix, url)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		schemas, entries := processPages(gatewayPrefix, pagesSuffix, pages)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}

		for key, value := range schemas {
			response, err := preparePayload(gatewayPrefix, "database", domain, created, key, value)
			if err != nil {
				framework.HandleFailure(w, err)
				return
			}
			log.Println("received database response persist payload:", response)
		}

		for key, value := range entries {
			response, err := preparePayload(gatewayPrefix, "storage", domain, created, key, value)
			if err != nil {
				framework.HandleFailure(w, err)
				return
			}
			log.Println("received storage response persist payload:", response)
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
	response, err := http.Get(gatewayPrefix + pagesSuffix + "?" + params.Encode())
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
	gatewayPrefix string,
	pagesSuffix string,
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
		go getEntries(ch, gatewayPrefix, pagesSuffix, page)
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

func preparePayload(
	gatewayPrefix string,
	functionName string,
	domain string,
	created time.Time,
	key otodom.SchemaKey,
	schema interface{},
) (response *http.Response, err error) {
	payload := framework.Entry{
		Created:       created,
		Domain:        domain,
		SchemaName:    key.SchemaName,
		SchemaVersion: key.SchemaVersion,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Println("error while marshalling Entry", err)
		return
	}
	response, err = http.Post(gatewayPrefix+"/"+functionName, "application/json", bytes.NewBuffer(raw))
	if err != nil {
		log.Println("error when seding", functionName, "persist request", err)
		return
	}
	return
}

func getEntries(ch chan otodom.CrawlingResponse, gatewayPrefix string, crawlerSuffix string, page otodom.Page) {
	var (
		data otodom.CrawlingResponse
	)

	log.Println("sending otodom crawler request for", page.URL)
	params := url.Values{}
	params.Add("url", page.URL)
	targetURL := gatewayPrefix + crawlerSuffix + "?" + params.Encode()
	response, err := http.Get(targetURL)
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

	log.Println("received response from crawler for", page.URL)

	ch <- data
	return
}
