package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
)

var kafkaversion = sarama.V0_10_1_0

type Consumer struct {
	groupID      string
	seed_brokers []string
	topics       []string
	exitChannel  chan bool

	client *cluster.Client

	KafkaStatus KafkaStatus
	config      *cluster.Config

	Chan           chan *Message
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

func NewConsumer(brokers []string, groupid string) Consumer {

	c := Consumer{}
	c.seed_brokers = brokers
	c.consumers = make(map[string]*cluster.Consumer)

	c.groupID = groupid

	return c
}

func (s *Consumer) Init(topics []string) error {
	var (
		kafka_status KafkaStatus
		err          error
	)

	kafka_status, s.topics, err = FetchKafkaMetadata(s.seed_brokers, topics)
	if err != nil {
		logger.Fatalf("error fetching metadata from broker: %v", err)
	}
	s.KafkaStatus = kafka_status

	s.Chan = make(chan *Message, 256)

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

func (s *Consumer) StartConsumingTopic() error {
	consumer, err := cluster.NewConsumerFromClient(s.client, s.groupID, s.topics)

	if err != nil {
		fmt.Printf("Error on StartConsumingTopic for topic %s: %+v\n", s.topics, err)
		return err
	}

	s.consumersMutex.Lock()
	//	s.consumers[topic] = consumer
	defer s.consumersMutex.Unlock()

	go func(notifications <-chan *cluster.Notification) {
		for notification := range notifications {
			// XXX: see sarama-cluster/consumer.go `subs` variable. rebalane might be throwing errors even though all seems to be working:
			// kafka server: The provided member is not known in the current generation
			// kafka server: In the middle of a leadership election, there is currently no leader for this partition and hence it is unavailable for writes.
			if len(notification.Claimed)+len(notification.Released)+len(notification.Current) > 0 {
				fmt.Printf("Notification: %+v\n", notification)
			}
		}
	}(consumer.Notifications())

	go func(in <-chan *sarama.ConsumerMessage, out chan<- *Message) {
		for message := range in {
			m := &Message{}
			m.Key = string(message.Key)
			m.Topic = message.Topic
			m.Partition = message.Partition
			m.Data = message.Value

			out <- m
		}
	}(consumer.Messages(), s.Chan)

	fmt.Printf("Started consuming topic %s\n", s.topics)

	return nil
}

func (s *Consumer) Wait() {
	<-s.exitChannel
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
func FetchKafkaMetadata(seed_brokers, topics []string) (KafkaStatus, []string, error) {
	kf := NewKafkaStatus()
	found := []string{}

	config := sarama.NewConfig()
	config.Version = kafkaversion
	broker, err := connectToBroker(config, seed_brokers)
	if err != nil {
		return kf, nil, err
	}
	request := sarama.MetadataRequest{}
	response, err := broker.GetMetadata(&request)
	if err != nil {
		_ = broker.Close()
		return kf, nil, err
	}

	if len(response.Brokers) == 0 {
		return kf, nil, errors.New(fmt.Sprintf("Unable to find any brokers"))
	}

	for _, broker := range response.Brokers {
		kf.Brokers = append(kf.Brokers, broker.Addr())
	}
	fmt.Printf("Brokers: %+v\n", kf.Brokers)

	regexpFilter := regexp.MustCompile(strings.Join(topics, "|"))
	for _, topic := range response.Topics {
		if regexpFilter.MatchString(topic.Name) {
			found = append(found, topic.Name)
			kf.Topics = append(kf.Topics, topic.Name)
			for _, partition := range topic.Partitions {
				kf.TopicPartitions[topic.Name] = append(kf.TopicPartitions[topic.Name], partition.ID)
			}
		}
	}
	return kf, found, broker.Close()
}
