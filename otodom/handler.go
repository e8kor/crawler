package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	Page int
}
type PageSorter []Page

func (a PageSorter) Len() int           { return len(a) }
func (a PageSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PageSorter) Less(i, j int) bool { return a[i].Page < a[j].Page }

func Handle(r handler.Request) (handler.Response, error) {
	var (
		response handler.Response
		entries  []Entry
	)

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
	for _, url := range urls {
		for _, page := range collectPages(url) {
			entries = append(entries, collectEntries(page.URL)...)
		}
	}

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

func collectEntries(url string) []Entry {
	var (
		entries []Entry
		c       = colly.NewCollector()
	)
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

	c.Visit(url)
	return entries
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}
