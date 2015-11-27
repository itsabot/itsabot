package dt

import (
	"encoding/json"
	"log"
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/mattbaird/elastigo/lib"
)

type SearchClient struct {
	*elastigo.Conn
}

type Bucket struct {
	Key      string
	DocCount uint `json:"doc_count"`
}

func NewSearchClient() *SearchClient {
	client := elastigo.NewConn()
	client.Username = os.Getenv("ELASTICSEARCH_USERNAME")
	client.Password = os.Getenv("ELASTICSEARCH_PASSWORD")
	client.Domain = os.Getenv("ELASTICSEARCH_DOMAIN")
	return &SearchClient{client}
}

func (ec *SearchClient) FindProducts(query, typ string, budget uint64,
	count int) ([]Product, error) {
	// JSON is the worst querying language ever
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							"Price": map[string]interface{}{
								"gte": budget - uint64(float64(budget)*0.2),
								"lte": budget + uint64(float64(budget)*0.2),
							},
						},
					},
					map[string]interface{}{
						"match": map[string]string{"_all": query},
					},
				},
			},
		},
	}
	log.Println("SEARCHING", typ, "FOR", query, "lte", budget+uint64(float64(budget)*0.2))
	res, err := ec.Search("products", typ, nil, q)
	if err != nil {
		return []Product{}, err
	}
	if res.Hits.Total == 0 {
		log.Println("NO RESULTS")
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
