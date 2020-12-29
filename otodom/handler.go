package function

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gocolly/colly/v2"
)

// Entry stores Otodom dashboard structure
type Entry struct {
	Title  string
	Name   string
	Region string
	Price  string
	Area   string
	Link   string
}

func Handle(w http.ResponseWriter, r *http.Request) {
	URL := r.URL.Query().Get("url")
	if URL == "" {
		URL = os.Getenv("SOURCE_URL")
	}
	if URL == "" {
		log.Fatalln("{ \"error\": \"missing url parameter\"}")
		w.Write([]byte("[]"))
	} else {
		c := colly.NewCollector()
		entries := make([]Entry, 0, 20)
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

		c.Visit(URL)
		enc := json.NewEncoder(w)
		enc.Encode(entries)
	}
}
