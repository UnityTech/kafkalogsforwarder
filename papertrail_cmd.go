package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/RackSec/srslog"
	"github.com/buger/jsonparser"
	"github.com/urfave/cli"
)

var papertrailprefix string

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
						jsondata.Ts, _ = jsonparser.GetUnsafeString(msg.Data, "ts")
						jsondata.Msg, _ = jsonparser.GetString(msg.Data, "log")
						jsondata.Level, _ = jsonparser.GetUnsafeString(msg.Data, "level")
						jsondata.Stream, _ = jsonparser.GetUnsafeString(msg.Data, "stream")
						jsondata.Service, _ = jsonparser.GetUnsafeString(msg.Data, "service")
						jsondata.Host, _ = jsonparser.GetUnsafeString(msg.Data, "host")
						jsondata.IPAddress, _ = jsonparser.GetUnsafeString(msg.Data, "ip_address")
						jsondata.ServerIP, _ = jsonparser.GetUnsafeString(msg.Data, "server_ip")
						jsondata.DockerImage, _ = jsonparser.GetUnsafeString(msg.Data, "docker_image")
						jsondata.ContainerName, _ = jsonparser.GetUnsafeString(msg.Data, "container_name")
						jsondata.ContainerID, _ = jsonparser.GetUnsafeString(msg.Data, "container_id")

						// NOTE: GetString does some allocations, which might cause some overhead.
						jsondata.Msg = strings.TrimSpace(jsondata.Msg)
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
	repository, tag := parseImage(d.DockerImage)
	if d.ServerIP == "" {
		d.ServerIP = d.IPAddress
	}
	if tag != "" {
		return fmt.Sprintf("%s|%s%s|%s@%s|%s", d.Ts, papertrailprefix, repository, tag, d.ServerIP, d.Msg)
	} else {
		return fmt.Sprintf("%s|%s%s|%s|%s", d.Ts, papertrailprefix, repository, d.ServerIP, d.Msg)
	}
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
		msg = fmt.Sprintf("<%d> %s %s %s[%d]: %s", p, parts[0], parts[1], parts[2], os.Getpid(), parts[3])
	} else {
		msg = srslog.DefaultFormatter(p, hostname, tag, content)
	}
	return
}
