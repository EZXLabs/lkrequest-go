package main

import (
	"fmt"
	"log"

	"github.com/EZ-XLabs/lkrequest-go/lkrequest"
)

func main() {
	client, err := lkrequest.NewClientBuilder().
		SetVerify(true).
		SetTimeoutTotal(10000).
		Build()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	session, err := lkrequest.NewSessionBuilder(client).
		SetMaxRedirects(5).
		SetMaxConnections(4).
		SetIdleTimeout(5000).
		SetRetryFixed(2, 200).
		Build()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	req, err := lkrequest.NewRequest(session, "GET", "https://httpbin.org/get")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := req.
		AddHeader("Accept", "application/json").
		AddQuery("source", "session-builder-example").
		Send()
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode())
	fmt.Printf("URL: %s\n", resp.URL())
	fmt.Printf("Body:\n%s\n", resp.String())
}
