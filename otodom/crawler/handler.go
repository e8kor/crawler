package function

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/gocolly/colly/v2"

	framework "github.com/e8kor/crawler/commons"
	otodom "github.com/e8kor/crawler/otodom/commons"
)

//Handle is main function entrypoint
func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		entries      []otodom.Entry
		response     otodom.CrawlingResponse
		httpResponse *http.Response
	)
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		framework.HandleFailure(w, err)
		return
	}

	var (
		urls           = query["url"]
		destenationURL = r.Header.Get("X-Callback-Url")
	)

	if urls == nil {
		log.Println("missing url parameter")
		return
	}
	for _, item := range urls {
		item = url.PathUnescape(item)
		entries = append(entries, collectEntries(item)...)
	}
	response = otodom.CrawlingResponse{
		SchemaName:    "otodom.rent",
		SchemaVersion: "v0.0.1",
		Schema: otodom.Schema{
			Title:      otodom.Field{"Title", "Advertisement Post title", "text"},
			Name:       otodom.Field{"Agency Name", "Agency name or Private Offer", "text"},
			Region:     otodom.Field{"Estate localtion", "Estate location in Poland", "text"},
			Price:      otodom.Field{"Price for square meter", "Price in square meter in Polish zloty", "zl/m2"},
			TotalPrice: otodom.Field{"Total estate price", "Total estate in Polish zloty", "zl"},
			Area:       otodom.Field{"Available area", "Available area in square meters", "m2"},
			Link:       otodom.Field{"URL", "Offer URL", "URL"},
		},
		Entries: entries,
	}

	if destenationURL != "" {
		log.Printf("using callback %s\n", destenationURL)
		raw, err := json.Marshal(response)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		httpResponse, err = http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
		if err != nil {
			return
		}
		log.Printf("received x-callback-url %s response: %v\n", destenationURL, httpResponse)
	}

	framework.HandleSuccess(w, response)
	return
}

func collectEntries(url string) (entries []otodom.Entry) {

	c := colly.NewCollector()

	c.OnHTML("article[id]", func(e *colly.HTMLElement) {
		entry := otodom.Entry{
			Title:      e.ChildText("div.offer-item-details > header > h3 > a > span > span"),
			Name:       e.ChildText("div.offer-item-details-bottom > ul > li.pull-right"),
			Region:     e.ChildText("div.offer-item-details > header > p"),
			Price:      e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-price-per-m"),
			TotalPrice: e.ChildText("div.offer-item-details > ul > li.offer-item-price"),
			Area:       e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-area"),
			Link:       e.ChildAttr("div.offer-item-details > header > h3 > a", "href"),
		}
		entries = append(entries, entry)
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	c.Visit(url)

	log.Printf("collected %d records for url %s\n", len(entries), url)
	return entries
}
