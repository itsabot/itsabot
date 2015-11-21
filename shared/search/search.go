// Package search finds items from Ava's repository of knowledge.
package search

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/mattbaird/elastigo/lib"
)

type ElasticClient struct {
	*elastigo.Conn
}

type Bucket struct {
	Key      string
	DocCount uint `json:"doc_count"`
}

func NewClient() *ElasticClient {
	client := elastigo.NewConn()
	client.Username = os.Getenv("ELASTICSEARCH_USERNAME")
	client.Password = os.Getenv("ELASTICSEARCH_PASSWORD")
	client.Domain = os.Getenv("ELASTICSEARCH_DOMAIN")
	return &ElasticClient{client}
}

func (ec *ElasticClient) FindProduct(query, typ string, count int) (
	[]datatypes.Product, error) {
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]string{"_all": query},
		},
	}
	res, err := ec.Search("products", typ, nil, q)
	if err != nil {
		return []datatypes.Product{}, err
	}
	if res.Hits.Total == 0 {
		return []datatypes.Product{}, nil
	}
	/*
		for _, hit := range res.Hits.Hits {
		}
	*/
	return []datatypes.Product{}, errors.New("FindProduct not implemented")
}

func (ec *ElasticClient) FindProductKeywords(typ string) ([]Bucket, error) {
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
