package commons

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
