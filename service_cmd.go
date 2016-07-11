package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands,
		cli.Command{
			Name:  "service",
			Usage: "tails logs for a single service",
			Action: func(c *cli.Context) {

				if len(c.Args()) > 0 && c.Args()[0] != "" {
					globalFlags.Topic = strings.Split(c.Args()[0], ",")
				}

				consumer := NewConsumer(globalFlags.Brokers, globalFlags.Groupid)
				consumer.Init(globalFlags.Topic)

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

							// Strings are escaped in kafka, unescape them here if possible:
							// Note: This does cause bunch of extra overhead. Would be better to do all this using []byte
							msg2, err := strconv.Unquote("`" + msg + "`")
							if err == nil {
								msg = strings.TrimSpace(msg2)
							}
							fmt.Printf("%s: %s %s %s\n", service, ts, level, msg)
						}
					}
				}(consumer.Chan)

				consumer.StartConsumingTopic()

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
