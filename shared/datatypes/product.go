package datatypes

// Product represents a product result returned from ElasticSearch. Note that
// because it's an ElasticSearch result, it has a string ID.
type Product struct {
	ID      string
	Name    string
	Price   uint64
	Reviews []struct {
		Score uint
		Body  string
	}
}
