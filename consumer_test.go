package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
	"time"
)

var Brokers = []string{"172.16.6.187:9092"}

func TestConsumer(t *testing.T) {

	_ = Consumer{}
	assert.Nil(t, nil)
}

func TestNewConsumer(t *testing.T) {

	c := NewConsumer(Brokers, "")

	fmt.Printf("consumer: %+v", c)

	//c.Start()


}

func TestFetchKafkaMetadata(t *testing.T) {

	kafka_status, err := FetchKafkaMetadata(Brokers, "")
	assert.Nil(t, err)

	fmt.Printf("topics: %+v\n", kafka_status.Topics)
	fmt.Printf("brokers: %+v\n", kafka_status.Brokers)

	//c.Start()
}


func TestStart(t *testing.T) {

	s := NewConsumer(Brokers, "")
	s.Init()

	err := s.StartConsumingTopic("test")
	assert.Nil(t, err)

	time.Sleep(time.Minute)

	//c.Start()
}

