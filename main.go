package main

import (
	"github.com/jrmycanady/nokiahealth"
	"net/http"
)

func main() {
	consumerKey := flag.String("consumer-key", "", "oauth consumer key")
	consumerSecret := flag.String("consumer-secret", "", "oauth consumer secret")
	fqdn := flag.String("fqdn", "localhost", "fqdn for oauth callbacks")

	flag.Parse()

	if *consumerKey == "" {
		log.Fatalf("consumer key must be provided")
	}
	if *consumerSecret == "" {
		log.Fatalf("consumer secret must be provided")
	}

	client := nokiahealth.NewClient("<consumer_key>", " <consumer_secret>", "<callback_url>")
}
