package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestElasticsearch(t *testing.T) {

	_ = Elasticsearch{}
	assert.Nil(t, nil)
	fmt.Printf("")
}
/*
func TestNewElasticsearch(t *testing.T) {

	es, err := NewElasticsearch("http://localhost:9200")
	assert.Nil(t, err)

	fmt.Printf("es: %+v\n", es)
}

*/