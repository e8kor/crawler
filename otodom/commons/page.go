package commons

// Page stores Otodom dashboard structure
type Page struct {
	URL  string `json:"url"`
	Page int    `json:"page"`
}

// PageSorter is API for Page collection
type PageSorter []Page

func (a PageSorter) Len() int           { return len(a) }
func (a PageSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PageSorter) Less(i, j int) bool { return a[i].Page < a[j].Page }
