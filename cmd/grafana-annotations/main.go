package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	grafana "github.com/grafana/grafana-api-golang-client"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	Verbose    bool     `short:"v" long:"verbose" description:"Print verbose information"`
	GrafanaURL string   `short:"g" long:"grafana-url" description:"URL of the MQTT broker" env:"GRAFANA_URL" required:"yes"`
	Tag        []string `short:"t" long:"tag" description:"tag to filter Grafana annotations. If speficied multiple times, they are ANDed."`
}

func main() {
	_, err := flags.Parse(&opts)

	if err != nil {
		os.Exit(1)
	}

	grafanaURL, err := url.Parse(opts.GrafanaURL)

	if err != nil {
		log.Fatal(err)
	}

	baseURL := grafanaURL.Scheme + "://" + grafanaURL.Host

	if opts.Verbose {
		log.Printf("Connecting to Grafana at %v\n", baseURL)
	}

	client, err := grafana.New(baseURL, grafana.Config{BasicAuth: grafanaURL.User})

	if err != nil {
		log.Fatal(err)
	}

	annotations, err := client.Annotations(url.Values{"tags": opts.Tag})

	if err != nil {
		log.Fatal(err)
	}

	for _, a := range annotations {
		fmt.Printf("%v: %v (%v)\n", time.Unix(a.Time/1000, a.Time%1000).Format(time.RFC3339), a.Text, strings.Join(a.Tags, ","))
	}
}
