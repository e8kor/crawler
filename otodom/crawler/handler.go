package function

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

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
func CollectEntries(url string) (entries []otodom.Entry) {

	c := colly.NewCollector()

	c.OnHTML("article[id]", func(e *colly.HTMLElement) {
		entry := otodom.Entry{
			Title:      e.ChildText("div.offer-item-details > header > h3 > a > span > span"),
			Name:       e.ChildText("div.offer-item-details-bottom > ul > li.pull-right"),
			Region:     e.ChildText("div.offer-item-details > header > p"),
			Price:      ExtractNumber(e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-price-per-m")),
			TotalPrice: ExtractNumber(e.ChildText("div.offer-item-details > ul > li.offer-item-price")),
			Area:       ExtractNumber(e.ChildText("div.offer-item-details > ul > li.hidden-xs.offer-item-area")),
			Link:       TakeChractersBefore(e.ChildAttr("div.offer-item-details > header > h3 > a", "href"), ".html"),
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

//ExtractNumber pattempt to parse number from string
func ExtractNumber(raw string) (number string) {
	pattern := regexp.MustCompile(`(\d+)`)
	items := pattern.FindAllStringSubmatch(raw, -1)
	for _, item := range items {
		number = number + item[1]
	}
	return number
}

//TakeChractersBefore take string before character
func TakeChractersBefore(raw string, predicate string) (result string) {
	return raw[:strings.Index(raw, predicate)]
}
