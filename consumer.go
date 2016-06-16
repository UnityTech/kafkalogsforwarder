package main

import (
	//    "strconv"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"strings"
	"sync"
)

type Consumer struct {
	groupID      string
	seed_brokers []string
	topic_prefix string
	exitChannel  chan bool

	client *cluster.Client

	KafkaStatus KafkaStatus
	config      *cluster.Config

	consumers      map[string]*cluster.Consumer
	consumersMutex sync.Mutex
}

type KafkaStatus struct {
	Brokers []string

	Topics []string
	// Map: Topic name -> []int32 slice of partitions
	TopicPartitions map[string][]int32
}

func NewKafkaStatus() KafkaStatus {
	c := KafkaStatus{}
	c.TopicPartitions = make(map[string][]int32)

	return c
}

func NewConsumer(brokers []string, topic_prefix string) Consumer {

	c := Consumer{}
	c.seed_brokers = brokers
	c.consumers = make(map[string]*cluster.Consumer)

	c.groupID = "kafka2elasticsearch"

	c.topic_prefix = topic_prefix

	return c
}

func (s *Consumer) Init() error {

	kafka_status, err := FetchKafkaMetadata(s.seed_brokers, s.topic_prefix)
	if err != nil {
		logger.Fatalf("error fetching metadata from broker: %v", err)
	}
	s.KafkaStatus = kafka_status

	s.config = cluster.NewConfig()
	s.config.Group.Return.Notifications = true

	client, err := cluster.NewClient(s.KafkaStatus.Brokers, s.config)
	if err != nil {
		logger.Fatalf("error creating new kafka client: %v", err)
		return err
	}
	s.client = client

	return nil
}

func (s *Consumer) StartConsumingTopic(topic string) error {
	consumer, err := cluster.NewConsumerFromClient(s.client, s.groupID, []string{topic})

	if err != nil {
		fmt.Printf("Error on StartConsumingTopic for topic %s: %+v\n", topic, err)
		return err
	}

	s.consumersMutex.Lock()
	s.consumers[topic] = consumer
	defer s.consumersMutex.Unlock()

	go func(notifications <-chan *cluster.Notification) {
		for notification := range notifications {
			fmt.Printf("Notification: %+v\n", notification)
		}
	}(consumer.Notifications())

	go func(messages <-chan *sarama.ConsumerMessage) {
		for message := range messages {
			fmt.Printf("Message: %+v\n", message)
		}
	}(consumer.Messages())

	fmt.Printf("Started consuming topic %s\n", topic)

	return nil
}

func (s *Consumer) Wait() {
	<-s.exitChannel
}

func ConsumePartition(consumer sarama.Consumer, topic string, partition int32, initialOffset int64, messages chan Message) sarama.PartitionConsumer {
	var pc sarama.PartitionConsumer
	pc, err := consumer.ConsumePartition(topic, partition, initialOffset)
	if err != nil {
		logger.Fatalln(err)
	}
	go func(pc sarama.PartitionConsumer, messages chan Message) {
		for {
			select {
			case err := <-pc.Errors():
				logger.Printf("Consumer error: %+v", err)
			case msg := <-pc.Messages():
				m := Message{}
				m.Key = string(msg.Key)
				m.Topic = msg.Topic
				m.Partition = msg.Partition
				m.Data = msg.Value

				if err != nil {
					logger.Printf("Invalid JSON entry in log feed: %s\n", string(msg.Value))
					continue
				}

				messages <- m
			}
		}

	}(pc, messages)

	return pc
}

// connects to one of a list of brokers
func connectToBroker(config *sarama.Config, seed_brokers []string) (*sarama.Broker, error) {
	var err error
	for _, host := range seed_brokers {
		broker := sarama.NewBroker(host)
		err = broker.Open(config)
		if err != nil {
			logger.Printf("error connecting to broker: %s %v", host, err)
		} else {
			return broker, nil
		}
	}
	return nil, err
}

// connects to the broker and fetches current brokers' address and partition ids
func FetchKafkaMetadata(seed_brokers []string, topic_search_prefix string) (KafkaStatus, error) {
	kf := NewKafkaStatus()

	config := sarama.NewConfig()
	broker, err := connectToBroker(config, seed_brokers)
	if err != nil {
		return kf, err
	}
	request := sarama.MetadataRequest{}
	response, err := broker.GetMetadata(&request)
	if err != nil {
		_ = broker.Close()
		return kf, err
	}

	if len(response.Brokers) == 0 {
		return kf, errors.New(fmt.Sprintf("Unable to find any brokers"))
	}

	for _, broker := range response.Brokers {
		// log.Printf("broker: %q", broker.Addr())
		kf.Brokers = append(kf.Brokers, broker.Addr())
	}
	fmt.Printf("Brokers: %+v\n", kf.Brokers)

	for _, topic := range response.Topics {
		if strings.HasPrefix(topic.Name, topic_search_prefix) {
			kf.Topics = append(kf.Topics, topic.Name)
			for _, partition := range topic.Partitions {
				kf.TopicPartitions[topic.Name] = append(kf.TopicPartitions[topic.Name], partition.ID)
			}
		}
	}
	return kf, broker.Close()
}
