package main

import (
	"encoding/json"
	"fmt"
	"github.com/RackSec/srslog"
	"github.com/buger/jsonparser"
	"github.com/urfave/cli"
	"log"
	"strconv"
	"strings"
	"time"
)

var papertrailprefix string

type PapertrailLog struct {
	Message       string  `json:"message,omitempty"`
	Component     string  `json:"component,omitempty"`
	Severity      string  `json:"severity,omitempty"`
	Action        string  `json:"action,omitempty"`
	Subject       string  `json:"subject,omitempty"`
	CorrelationId string  `json:"correlation_id,omitempty"`
	MessageId     string  `json:"message_id,omitempty"`
	Name          string  `json:"name,omitempty"`
	Error         string  `json:"error,omitempty"`
	Errors        string  `json:"errors,omitempty"`
	Stack         string  `json:"stack,omitempty"`
	State         string  `json:"state,omitempty"`
	Time          int64   `json:"time,omitempty"`
	DurationS     float64 `json:"duration_s,omitempty"`
}

func init() {
	app.Commands = append(app.Commands,
		cli.Command{
			Name:  "papertrail",
			Usage: "forward logs to papertrail",
			Action: func(c *cli.Context) {

				papertrailprefix = c.String("prefix")
				consumer := NewConsumer(globalFlags.Brokers, globalFlags.Groupid)
				consumer.Init(globalFlags.Topic)

				log.Println(c.String("papertrail"))

				go listenforchecks()
				papertrail := make(chan *data, 20)
				defer close(papertrail)
				go func(messages <-chan *Message, papertrail chan<- *data) {
					for msg := range messages {
						jsondata := &data{start: time.Now(), offset: msg.Offset}
						jsondata.Ts, _ = jsonparser.GetUnsafeString(msg.Data, "time")
						jsondata.Msg, _ = jsonparser.GetUnsafeString(msg.Data, "log")
						jsondata.Level, _ = jsonparser.GetUnsafeString(msg.Data, "level")
						jsondata.Service, _ = jsonparser.GetUnsafeString(msg.Data, "kubernetes", "labels", "app")
						jsondata.Environment, _ = jsonparser.GetUnsafeString(msg.Data, "kubernetes", "labels", "environment")
						jsondata.Host, _ = jsonparser.GetUnsafeString(msg.Data, "kubernetes", "host")
						jsondata.IPAddress, _ = jsonparser.GetUnsafeString(msg.Data, "ip_address")
						jsondata.ServerIP, _ = jsonparser.GetUnsafeString(msg.Data, "server_ip")
						jsondata.DockerImage, _ = jsonparser.GetUnsafeString(msg.Data, "docker_image")
						jsondata.ContainerName, _ = jsonparser.GetUnsafeString(msg.Data, "kubernetes", "pod_name")
						jsondata.ContainerID, _ = jsonparser.GetUnsafeString(msg.Data, "docker", "container_id")

						// Just send the original data as the message if it for some reason doesn't have anything in the log field.
						if jsondata.Msg == "" {
							var papertrailLog PapertrailLog
							papertrailLog.Message, _ = jsonparser.GetUnsafeString(msg.Data, "message")
							papertrailLog.Component, _ = jsonparser.GetUnsafeString(msg.Data, "component")
							papertrailLog.Severity, _ = jsonparser.GetUnsafeString(msg.Data, "severity")
							papertrailLog.Action, _ = jsonparser.GetUnsafeString(msg.Data, "action")
							papertrailLog.Subject, _ = jsonparser.GetUnsafeString(msg.Data, "subject")
							papertrailLog.CorrelationId, _ = jsonparser.GetUnsafeString(msg.Data, "correlation_id")
							papertrailLog.MessageId, _ = jsonparser.GetUnsafeString(msg.Data, "message_id")
							papertrailLog.Name, _ = jsonparser.GetUnsafeString(msg.Data, "name")
							papertrailLog.Error, _ = jsonparser.GetUnsafeString(msg.Data, "error")
							papertrailLog.Errors, _ = jsonparser.GetUnsafeString(msg.Data, "errors")
							papertrailLog.Stack, _ = jsonparser.GetUnsafeString(msg.Data, "stack")
							papertrailLog.DurationS, _ = jsonparser.GetFloat(msg.Data, "duration_s")
							papertrailLog.Time, _ = jsonparser.GetInt(msg.Data, "time")
							papertraiLogJson, _ := json.Marshal(papertrailLog)
							jsondata.Msg = string(papertraiLogJson)
						}

						papertrail <- jsondata
					}
				}(consumer.Chan, papertrail)
				go Sender(papertrail, c.String("papertrail"), c.String("cert"))

				consumer.StartConsumingTopic()

				consumer.Wait()
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "papertrail",
					Usage:  "The papertrail address where the logs should be forwarded",
					EnvVar: "PAPERTRAIL_ADDRESS",
				},
				cli.StringFlag{
					Name:   "cert",
					Usage:  "Papertrail root certificate path",
					Value:  "./papertrail-bundle.pem",
					EnvVar: "PAPERTRAIL_CERT",
				},
				cli.StringFlag{
					Name:   "prefix",
					Usage:  "String that is prepended to the system name",
					EnvVar: "PAPERTRAIL_PREFIX",
				},
			},
		})
}

type data struct {
	Ts            string `json:"ts"`
	Msg           string `json:"msg"`
	Level         string `json:"level"`
	Stream        string `json:"stream"`
	Service       string `json:"service"`
	Environment   string `json:"environment"`
	Host          string `json:"host"`
	IPAddress     string `json:"ip_address"` // legacy
	ServerIP      string `json:"server_ip"`
	DockerImage   string `json:"docker_image"`
	ContainerName string `json:"container_name"`
	ContainerID   string `json:"container_id"`

	start  time.Time
	offset int64
}

// Sender receives valid entries from chan 'c' and uploads them into papertrail over an encrypted TCP connection.
func Sender(c <-chan *data, address, cert string) {
	if len(address) == 0 {
		for s := range c {
			fmt.Println(s)
		}
		return
	}

	w, err := srslog.DialWithTLSCertPath("tcp+tls", address, srslog.LOG_INFO, "kafkatopapertrail", cert)
	if err != nil {
		log.Fatal(err)
	}

	var timechan chan time.Duration
	if globalFlags.Verbose {
		timechan = make(chan time.Duration, 10)
		defer close(timechan)
		go func(timechan chan time.Duration) {
			var (
				eventcount int
				counter    time.Duration
				ticker     = time.NewTicker(time.Second * 30)
			)

		loop:
			for {
				select {
				case duration, ok := <-timechan:
					if ok {
						counter += duration
						eventcount++
					} else {
						break loop
					}
				case <-ticker.C:
					if eventcount != 0 {
						counter = counter / time.Duration(eventcount)
					}
					log.Printf("Sent %d messages during the last 30 seconds, averaging %s per message\n", eventcount, counter)
					eventcount = 0
					counter = 0
				}
			}
		}(timechan)
	}

	w.SetFormatter(LogFormatter)
	for msg := range c {
		switch msg.Level {
		case "INFO":
			err = w.Info(msg.String())
		case "ALERT":
			err = w.Alert(msg.String())
		case "CRIT", "CRITICAL":
			err = w.Crit(msg.String())
		case "DEBUG":
			err = w.Debug(msg.String())
		case "EMERG":
			err = w.Emerg(msg.String())
		case "ERR", "ERROR":
			err = w.Err(msg.String())
		case "NOTICE":
			err = w.Notice(msg.String())
		case "WARNING", "WARN":
			err = w.Warning(msg.String())
		default:
			err = w.Info(msg.String())
			//      log.Printf("Unknown log level: %s\n", msg.Level)
		}
		if err != nil {
			log.Println(err)
		}
		if globalFlags.Verbose {
			timechan <- time.Since(msg.start)
		}
	}
}

func (d *data) String() string {
	prefix := papertrailprefix
	ts, _ := strconv.ParseInt(d.Ts, 10, 64)
	timestamp := time.Unix(ts, 0).Format(time.RFC3339)
	if d.Environment == "stg" {
		// A bit conflicting with the whole point of shared prefix but mostly a easy solution until some things are sorted.
		prefix = "staging-"
	}
	return fmt.Sprintf("%s|%s%s|%s|%s %s", timestamp, prefix, d.Service, d.ContainerName, timestamp, d.Msg)
}

func parseImage(image string) (repository, tag string) {
	// Docker image format is:
	// {registry}/(_|/r/{user_or_org})/{repository}:{tag}
	// Here we get the repository and tag from that format
	var (
		parts []string
	)
	parts = strings.Split(image, "/")
	image = parts[len(parts)-1]
	parts = strings.Split(image, ":")
	repository = parts[0]
	if len(parts) > 1 {
		tag = parts[1]
		if len(tag) > 8 {
			tag = tag[:8]
		}
	}
	return
}

// LogFormatter is a custom syslog formatter that uses log data to fill timestamp, hostname and tag instead of using local system information.
func LogFormatter(p srslog.Priority, hostname, tag, content string) (msg string) {
	parts := strings.SplitN(content, "|", 4)
	if len(parts) == 4 {
		msg = fmt.Sprintf("<%d>1 %s %s %s - - - %s", p, parts[0], parts[1], parts[2], parts[3])
	} else {
		msg = srslog.DefaultFormatter(p, hostname, tag, content)
	}
	return
}
