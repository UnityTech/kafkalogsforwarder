package main

import (
    "strconv"
    "strings"
    "gopkg.in/Shopify/sarama.v1"
    "errors"
    "fmt"
)

type Consumer struct {
    config *sarama.Config
    brokers []string
    topic string
    exitChannel chan bool
    consumer sarama.Consumer
    partitionConsumers []sarama.PartitionConsumer

}

func (s *Consumer) Start(brokers []string, topic string, partition string, channel chan Message) {

    var initialOffset  int64
    var err error
    var partitions []int32

    s.exitChannel = make(chan bool)
    s.brokers = brokers
    s.topic = topic
    initialOffset = sarama.OffsetNewest


    s.config = sarama.NewConfig()
    s.config.ClientID = "kafkalogs"

    all_brokers, all_partitions, err := s.fetchMetadata(s.config)
    if err != nil {
        logger.Fatalf("error fetching metadata from broker: %v", err)
    }


    if s.consumer, err = sarama.NewConsumer(all_brokers, s.config); err != nil {
        logger.Fatalln(err)
    }

    if partition != "" {
        partitions = []int32{}
        for _, v := range strings.Split(partition, ",") {
            if i, err := strconv.Atoi(v); err == nil {
                partitions = append(partitions, int32(i))
            }
        }
    } else {
        partitions = all_partitions
    }

    if len(partitions) == 0 {
        logger.Fatalln("No partitions listed")
    } else {
        logger.Printf("Consuming topic %s for partitions: %v\n", topic, partitions)
    }

    for _, partition := range partitions {
        pc := ConsumePartition(s.consumer, topic, partition, initialOffset, channel)
        s.partitionConsumers = append(s.partitionConsumers, pc)
    }

}

// WaitForExit waits for os signal, then closes the sarama connection.
func (s *Consumer) Close() {
    for _, pc := range s.partitionConsumers {
        pc.AsyncClose()
    }

    s.consumer.Close();
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
func (c *Consumer) connectToBroker(config *sarama.Config) (*sarama.Broker, error) {
    var err error
    for _, host := range c.brokers {
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
func (c *Consumer) fetchMetadata(config *sarama.Config) ([]string, []int32, error) {
    broker, err := c.connectToBroker(config)
    if err != nil {
        return nil, nil, err
    }
    request := sarama.MetadataRequest{Topics: []string{c.topic}}
    response, err := broker.GetMetadata(&request)
    if err != nil {
        _ = broker.Close()
        return nil, nil, err
    }

    if len(response.Brokers) == 0 {
        return nil, nil, errors.New(fmt.Sprintf("Unable to find any broker for topic: %s", c.topic))
    }
    if len(response.Topics) != 1 {
        return nil, nil, errors.New(fmt.Sprintf("Invalid number of topics: %d", len(response.Topics)))
    }

    var brokers []string
    for _, broker := range response.Brokers {
        // log.Printf("broker: %q", broker.Addr())
        brokers = append(brokers, broker.Addr())
    }

    var partitions []int32
    for _, partition := range response.Topics[0].Partitions {
        //logger.Printf("partition: %v, leader: %v", partition.ID, partition.Leader)
        partitions = append(partitions, partition.ID)
    }

    return brokers, partitions, broker.Close()
}