package main

import (
    "fmt"
    "github.com/codegangsta/cli"
    "strings"
    "os"
)

func init() {
    app.Commands = append(app.Commands,
        cli.Command{
            Name:  "firehose",
            Usage: "complete firehose of all log messages",
            Action: func(c *cli.Context) {

                consumer := Consumer{}
                incomingMessages := make(chan Message, 256)

                go func(messages chan Message) {
                    count := 0
                    for msg := range messages {
                        count++

                        if count % 100 == 0 {
                            if len(messages) > 100 {
                                fmt.Fprintf(os.Stderr, "Display goroutine is not fast enough: %d", len(messages))
                            }
                        }

                        msg.ParseJSON()

                        if c.Bool("all") {
                            fmt.Printf("%+v\n", msg.Container)
                        } else {

                            msg.ParseJSON()
                            ts, _ := msg.GetString("ts")
                            level, _ := msg.GetString("level")
                            msg, _ := msg.GetString("msg")
                            msg = strings.TrimRight(msg, "\n")
                            fmt.Printf("%s %s %s\n", ts, level, msg)
                        }
                    }
                }(incomingMessages)

                consumer.Start(globalFlags.Brokers, globalFlags.Topic, globalFlags.Partitions, incomingMessages)

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
