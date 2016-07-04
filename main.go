package main

import (
	"log"
	"os"
	"strings"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

var (
	logger             = log.New(os.Stderr, "", log.LstdFlags)
	app       *cli.App = cli.NewApp()
	builddate string
)

var globalFlags = struct {
	Brokers []string
	Topic   []string
	Offset  string
	Groupid string
	Verbose bool
}{}

func main() {
	app.Name = "kafkalogs"
	app.Usage = "kafkalogs is a tool to tail a kafka stream for json based log entries"
	app.Version = builddate

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "brokers",
			Usage:  "The comma separated list of brokers in the Kafka cluster",
			EnvVar: "LOGTAIL_KAFKA_BROKERS",
		},
		cli.StringFlag{
			Name:   "topic",
			Usage:  "Kafka topic to listen",
			EnvVar: "LOGTAIL_KAFKA_TOPIC",
		},
		cli.StringFlag{
			Name:   "groupid",
			Usage:  "Consumer group identifier",
			EnvVar: "LOGTAIL_KAFKA_GROUPID",
			Value:  uuid.NewV4().String(),
		},
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "Use verbose logging",
			EnvVar: "VERBOSE",
		},
	}

	app.Before = func(c *cli.Context) error {

		globalFlags.Brokers = strings.Split(c.String("brokers"), ",")
		globalFlags.Topic = strings.Split(c.String("topic"), ",")
		globalFlags.Offset = c.String("offiset")
		globalFlags.Groupid = c.String("groupid")
		globalFlags.Verbose = c.Bool("verbose")

		logger.Printf("Logtail starting with options %+v\n", globalFlags)

		return nil
	}

	app.Run(os.Args)
}
