package function

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	framework "github.com/e8kor/crawler/commons"
	otodom "github.com/e8kor/crawler/otodom/commons"

	"github.com/gocolly/colly/v2"
)

//Handle is main function entrypoint
func Handle(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	log.Println("preparing pages for", url)
	pages := collectPages(url)
	framework.HandleSuccess(w, pages)
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
		log.Println("searching for last page on", r.URL.String())
	})

	c.Visit(url)

	log.Println("found last page", lastPage)

	for i := 1; i < lastPage.Page; i++ {
		var pageURL string
		var index = strconv.Itoa(i)
		if strings.Contains(url, "?") {
			pageURL = url + "&page=" + index
		} else {
			pageURL = url + "?page=" + index
		}
		pages = append(pages, otodom.Page{
			Page: i,
			URL:  pageURL,
		})
	}

	log.Println("found", len(pages), "pages")

	return
}
