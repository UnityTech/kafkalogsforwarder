package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands,
		cli.Command{
			Name:  "service",
			Usage: "tails logs for a single service",
			Action: func(c *cli.Context) {

				consumer := NewConsumer(globalFlags.Brokers, "", globalFlags.Groupid)
				consumer.Init()

				go func(messages <-chan *Message) {
					count := 0
					for msg := range messages {

						count++
						if count%100 == 0 {
							if len(messages) > 100 {
								fmt.Fprintf(os.Stderr, "Display goroutine is not fast enough: %d\n", len(messages))
							}
							fmt.Fprintf(os.Stdout, "%d\r", count)
						}

						if c.Bool("all") {
							fmt.Printf("%s\n", msg.Data)
						} else {
							msg.ParseJSON()
							ts, _ := msg.GetString("ts")
							service, _ := msg.GetString("service")
							level, _ := msg.GetString("level")
							msg, _ := msg.GetString("msg")
							msg = strings.TrimRight(msg, "\n")
							fmt.Printf("%s: %s %s %s\n", service, ts, level, msg)
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
