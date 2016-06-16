package main

import "gopkg.in/olivere/elastic.v3"

type Elasticsearch struct {
	Client *elastic.Client
}

func NewElasticsearch(host string) (Elasticsearch, error) {

	e := Elasticsearch{}

	c, err := elastic.NewClient(elastic.SetURL(host), elastic.SetMaxRetries(10))
	if err != nil {
		return e, err
	}

	e.Client = c
	return e, nil
}