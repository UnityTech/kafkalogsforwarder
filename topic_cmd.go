package main

import (
    "fmt"
    "github.com/codegangsta/cli"
    "strings"
)

func init() {
    app.Commands = append(app.Commands,
        cli.Command{
            Name:  "topic",
            Usage: "tails a single topic",
            Action: func(c *cli.Context) {

                topic := c.Args()[0]

                consumer := Consumer{}
                incomingMessages := make(chan Message)

                go func(messages chan Message) {
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
                }(incomingMessages

                consumer.Start(globalFlags.Brokers, topic, globalFlags.Partitions, incomingMessages)

                consumer.Wait()
            },
            Flags: []cli.Flag{
                cli.BoolFlag{
                    Name:"all",
                    Usage:"Print entire JSON, not just level and msg fields",
                },
            },
        })
}
