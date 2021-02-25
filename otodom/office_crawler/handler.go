package function

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gocolly/colly/v2"

	framework "github.com/e8kor/crawler/commons"
	otodom "github.com/e8kor/crawler/otodom/commons"
)

// Entry stores Otodom dashboard structure
type Entry struct {
	Title      string `json:"title"`
	Name       string `json:"name"`
	Region     string `json:"region"`
	Price      string `json:"price"`
	TotalPrice string `json:"total_price"`
	Area       string `json:"area"`
	Link       string `json:"link"`
}

// Schema stores Otodom schema
type Schema struct {
	Title      otodom.Field `json:"title"`
	Name       otodom.Field `json:"name"`
	Region     otodom.Field `json:"region"`
	Price      otodom.Field `json:"price"`
	TotalPrice otodom.Field `json:"total_price"`
	Area       otodom.Field `json:"area"`
	Link       otodom.Field `json:"link"`
}

//Handle is main function entrypoint
func Handle(w http.ResponseWriter, r *http.Request) {
	var (
		entries      []interface{}
		response     otodom.CrawlingResponse
		httpResponse *http.Response
	)

	var (
		schemaName     = os.Getenv("SCHEMA_NAME")
		schemaVersion  = os.Getenv("SCHEMA_VERSION")
		item           = r.URL.Query().Get("url")
		destenationURL = r.Header.Get("X-Callback-Url")
	)

	entries = append(entries, CollectEntries(item)...)

	response = otodom.CrawlingResponse{
		SchemaName:    schemaName,
		SchemaVersion: schemaVersion,
		Schema: Schema{
			Title:      otodom.Field{"Title", "Advertisement Post title", "text"},
			Name:       otodom.Field{"Agency Name", "Agency name or Private Offer", "text"},
			Region:     otodom.Field{"Estate localtion", "Estate location in Poland", "text"},
			Price:      otodom.Field{"Price for square meter", "Price in square meter in Polish zloty", "number"},
			TotalPrice: otodom.Field{"Total estate price", "Total estate in Polish zloty", "number"},
			Area:       otodom.Field{"Available area", "Available area in square meters", "number"},
			Link:       otodom.Field{"URL", "Offer URL", "URL"},
		},
		Entries: entries,
	}

	if destenationURL != "" {
		log.Println("using callback:", destenationURL)
		raw, err := json.Marshal(response)
		if err != nil {
			framework.HandleFailure(w, err)
			return
		}
		httpResponse, err = http.Post(destenationURL, "application/json", bytes.NewBuffer(raw))
		if err != nil {
			return
		}
		log.Println("received x-callback-url", destenationURL, "response:", httpResponse)
	}

	framework.HandleSuccess(w, response)
	return
}

// CollectEntries crawls Otodom dashboard entries from url
func CollectEntries(url string) (entries []interface{}) {

	c := colly.NewCollector()

	c.OnHTML("article[id]", func(e *colly.HTMLElement) {
		entry := Entry{
			Title:      e.ChildText("div.offer-item-details > header > h3 > a > span > span"),
			Name:       e.ChildText("div.offer-item-details-bottom > ul > li.pull-right"),
			Region:     e.ChildText("div.offer-item-details > header > p"),
			Price:      otodom.ExtractNumber(e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-price-per-m")),
			TotalPrice: otodom.ExtractNumber(e.ChildText("div.offer-item-details > ul > li.offer-item-price")),
			Area:       otodom.ExtractNumber(e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-area")),
			Link:       otodom.TakeChractersBefore(e.ChildAttr("div.offer-item-details > header > h3 > a", "href"), ".html"),
		}
		entries = append(entries, entry)
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	c.Visit(url)

	log.Println("collected", len(entries), "records for url:", url)
	return entries
}
