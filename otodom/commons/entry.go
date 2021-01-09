package commons

// Field stores Otodom schema field
type Field struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Schema stores Otodom schema
type Schema struct {
	Title      Field `json:"title"`
	Name       Field `json:"name"`
	Region     Field `json:"region"`
	Price      Field `json:"price"`
	TotalPrice Field `json:"total_price"`
	Area       Field `json:"area"`
	Link       Field `json:"link"`
}

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

// CrawlingResponse stores Otodom schema
type CrawlingResponse struct {
	SchemaName    string  `json:"schema_name"`
	SchemaVersion string  `json:"schema_version"`
	Schema        Schema  `json:"schema"`
	Entries       []Entry `json:"entries"`
}

// SchemaKey store key for schema
type SchemaKey struct {
	SchemaName    string `json:"schema_name"`
	SchemaVersion string `json:"schema_version"`
}
