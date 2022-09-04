package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	grafana "github.com/grafana/grafana-api-golang-client"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	Verbose      bool     `short:"v" long:"verbose" description:"Print verbose information"`
	MqttClientID string   `short:"c" long:"client-id" description:"client id to use for the MQTT connection"`
	MqttURL      string   `short:"m" long:"mqtt-url"  description:"URL of the MQTT broker incl. username and password" env:"MQTT_URL" required:"yes"`
	GrafanaURL   string   `short:"g" long:"grafana-url" description:"URL of the Grafana server incl. username and password" env:"GRAFANA_URL" required:"yes"`
	Topics       []string `short:"t" long:"topic" description:"MQTT topic to subscribe to and publish to Grafana (repeat for multiple topics)" required:"yes"`
	Tag          []string `long:"tag" description:"tag to add to the Grafana annotation (repeat for multiple tags)"`
}

func main() {
	logger := log.New(os.Stderr, "", 0)

	_, err := flags.Parse(&opts)

	if err != nil {
		os.Exit(1)
	}

	grafanaURL, err := url.Parse(opts.GrafanaURL)

	if err != nil {
		logger.Fatal(err)
	}

	baseURL := grafanaURL.Scheme + "://" + grafanaURL.Host

	if opts.Verbose {
		logger.Printf("Connecting to Grafana at %v\n", baseURL)
	}

	client, err := grafana.New(baseURL, grafana.Config{BasicAuth: grafanaURL.User})

	if err != nil {
		logger.Fatal(err)
	}

	mqttURL, err := url.Parse(opts.MqttURL)

	if err != nil {
		logger.Fatal(err)
	}

	mqttClientID := opts.MqttClientID

	if mqttClientID == "" {
		mqttClientID = getProgramName()
	}

	mqttOpts := mqtt.NewClientOptions().
		AddBroker(mqttURL.String()).
		SetClientID(mqttClientID).
		SetCleanSession(false).
		SetUsername(mqttURL.User.Username()).
		SetAutoReconnect(true)

	mqttOpts.OnConnect = func(mqttClient mqtt.Client) {
		if opts.Verbose {
			logger.Printf("Connected to MQTT at %v\n", mqttURL.Host)
		}

		if opts.Verbose {
			logger.Printf("Subscribing to %v\n", strings.Join(opts.Topics, ","))
		}

		token := mqttClient.SubscribeMultiple(topicMap(opts.Topics), func(c mqtt.Client, m mqtt.Message) {
			text := m.Topic() + ": " + string(m.Payload())

			if opts.Verbose {
				logger.Printf("Publishing Grafana annotation: %v (%v)\n", text, strings.Join(opts.Tag, ","))
			}

			_, err := client.NewAnnotation(&grafana.Annotation{
				Text: text,
				Tags: opts.Tag,
			})

			if err != nil {
				logger.Printf("Error: could not publish annotation %v: %v", m.Payload(), err)
			}
		})

		if !token.WaitTimeout(10 * time.Second) {
			logger.Fatalf("Could not subscribe: %v", token.Error())
		}
	}

	mqttOpts.OnReconnecting = func(client mqtt.Client, options *mqtt.ClientOptions) {
		if opts.Verbose {
			logger.Printf("Reconnecting to MQTT at %s\n", mqttURL.String())
		}
	}

	password, isSet := mqttURL.User.Password()

	if isSet {
		mqttOpts.SetPassword(password)
	}

	if opts.Verbose {
		mqtt.WARN = log.New(os.Stderr, "WARN ", 0)
	}

	mqtt.CRITICAL = log.New(os.Stderr, "CRITICAL ", 0)
	mqtt.ERROR = log.New(os.Stderr, "ERROR ", 0)

	mqttClient := mqtt.NewClient(mqttOpts)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("Could not connect to MQTT: %s", token.Error())
	}

	_, err = client.NewAnnotation(&grafana.Annotation{
		Text: fmt.Sprintf("%v starting up", mqttClientID),
		Tags: opts.Tag,
	})

	if err != nil {
		logger.Printf("could not publish startup annotation: %v", err)
	}

	quitProgram := make(chan struct{})
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		_, err := client.NewAnnotation(&grafana.Annotation{
			Text: fmt.Sprintf("%v shutting down", mqttClientID),
			Tags: opts.Tag,
		})

		if err != nil {
			logger.Printf("could not publish shutdown annotation: %v", err)
		}

		mqttClient.Disconnect(250)
		close(quitProgram)
	}()

	<-quitProgram
}

func getProgramName() string {
	path, err := os.Executable()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: Could not determine program name; using 'unknown'.")
		return "unknown"
	}

	return filepath.Base(path)
}

func topicMap(topics []string) map[string]byte {
	tm := make(map[string]byte)

	for _, t := range topics {
		tm[t] = 0
	}

	return tm
}
