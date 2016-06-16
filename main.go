package main

import (
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

var (
	logger             = log.New(os.Stderr, "", log.LstdFlags)
	app       *cli.App = cli.NewApp()
	builddate string
)

var globalFlags = struct {
	Brokers    []string
	Topic      string
	Partitions string
	Offset     string
	Verbose    bool
}{}

func main() {
	app.Name = "kafkalogs"
	app.Usage = "kafkalogs is a tool to tail a kafka stream for json based log entries"
	app.Version = builddate

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "brokers",
			Usage:  "The comma separated list of brokers in the Kafka cluster",
			EnvVar: "LOGTAIL_KAFKA_CONNECTION_STRING",
		},
		cli.StringFlag{
			Name:   "topic",
			Usage:  "Kafka topic to listen",
			EnvVar: "LOGTAIL_KAFKA_TOPIC",
		},
		cli.StringFlag{
			Name:   "partitions",
			Usage:  "Comma separated list of partitions. Leave empty to listen all partitions",
			EnvVar: "PARTITIONS",
		},
		cli.StringFlag{
			Name:   "service",
			Usage:  "Filter only selected services",
			EnvVar: "SERVICE",
		},
	}

	app.Before = func(c *cli.Context) error {

		globalFlags.Brokers = strings.Split(c.String("brokers"), ",")
		globalFlags.Topic = c.String("topic")
		globalFlags.Partitions = c.String("partitions")
		globalFlags.Offset = c.String("offiset")

		logger.Printf("Logtail starting with options %+v\n", globalFlags)

		return nil
	}

	app.Run(os.Args)
}
