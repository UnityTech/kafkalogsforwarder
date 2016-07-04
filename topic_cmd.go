package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands,
		cli.Command{
			Name:  "topic",
			Usage: "tails a single topic",
			Action: func(c *cli.Context) {

				globalFlags.Topic = strings.Split(c.Args()[0], ",")

				consumer := NewConsumer(globalFlags.Brokers, "", globalFlags.Groupid)
				consumer.Init()

				go func(messages chan *Message) {
					for msg := range messages {

						if c.Bool("all") {
							fmt.Printf("%s\n", msg.Data)
						} else {
							ts, _ := msg.GetString("ts")
							level, _ := msg.GetString("level")
							msg, _ := msg.GetString("msg")
							msg = strings.TrimRight(msg, "\n")
							fmt.Printf("%s %s %s\n", ts, level, msg)
						}
					}
				}(consumer.Chan)

				consumer.StartConsumingTopic(globalFlags.Topic...)

				consumer.Wait()
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all",
					Usage: "Print entire JSON, not just level and msg fields",
				},
			},
		})
}
