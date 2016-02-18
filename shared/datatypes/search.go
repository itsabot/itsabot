package dt

import (
	"encoding/json"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/mattbaird/elastigo/lib"
)

// SearchClient wraps an ElasticSearch client connection to provide a higher
// level API in searching for products, etc.
type SearchClient struct {
	*elastigo.Conn
}

// Bucket is an ElasticSearch data structure that's used to get a count of
// ElasticSearch documents that match a given query.
type Bucket struct {
	Key      string
	DocCount uint `json:"doc_count"`
}

// NewSearchClient returns the higher-level API ElasticSearch connection with
// filled in auth data from ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD,
// and ELASTICSEARCH_DOMAIN environment variables.
func NewSearchClient() *SearchClient {
	client := elastigo.NewConn()
	client.Username = os.Getenv("ELASTICSEARCH_USERNAME")
	client.Password = os.Getenv("ELASTICSEARCH_PASSWORD")
	client.Domain = os.Getenv("ELASTICSEARCH_DOMAIN")
	return &SearchClient{client}
}

// FindProducts retrieves a slice of products for a given query, product
// category, product type and budget.
func (ec *SearchClient) FindProducts(query, category, typ string,
	budget uint64) ([]Product, error) {

	// JSON is the worst querying language ever
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							"Price": map[string]interface{}{
								"gte":   budget - uint64(float64(budget)*0.3),
								"lte":   budget + uint64(float64(budget)*0.3),
								"boost": 2.0,
							},
						},
					},
					map[string]interface{}{
						"match": map[string]string{"_all": query},
					},
					map[string]interface{}{
						"match": map[string]string{
							"Category": category,
						},
					},
				},
			},
		},
	}
	res, err := ec.Search("products", typ, nil, q)
	if err != nil {
		return []Product{}, err
	}
	if res.Hits.Total == 0 {
		log.WithFields(log.Fields{
			"q":      query,
			"cat":    category,
			"budget": budget,
		}).Infoln("no results")
		return []Product{}, nil
	}
	var products []Product
	for _, hit := range res.Hits.Hits {
		var prod Product
		err = json.Unmarshal([]byte(*hit.Source), &prod)
		if err != nil {
			return products, err
		}
		prod.ID = hit.Id
		products = append(products, prod)
	}
	return products, nil
}

// FindProductKeywords returns the counts of the most-common descriptive words
// in reviews for a product category. These most common words can then be
// labeled for the Summarization algorithm.
func (ec *SearchClient) FindProductKeywords(typ string) ([]Bucket, error) {
	q := map[string]interface{}{
		"aggs": map[string]interface{}{
			"keywords": map[string]interface{}{
				"terms": map[string]interface{}{
					"field":         "Reviews.Body",
					"size":          2500,
					"min_doc_count": 3,
				},
			},
		},
	}
	res, err := ec.Search("products", typ, nil, q)
	if err != nil {
		return []Bucket{}, err
	}
	var aggs struct {
		Keywords struct {
			Buckets []Bucket
		}
	}
	err = json.Unmarshal([]byte(res.Aggregations), &aggs)
	if err != nil {
		return []Bucket{}, err
	}
	return aggs.Keywords.Buckets, nil
}
