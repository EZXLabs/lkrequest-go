package main

import (
	"fmt"
	"io"
	"log"

	"github.com/EZXLabs/lkrequest-go/lkrequest"
)

func main() {
	client, err := lkrequest.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	session, err := lkrequest.NewSession(client)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	req, err := lkrequest.NewRequest(session, "GET", "https://httpbin.org/stream/5")
	if err != nil {
		log.Fatal(err)
	}

	stream, err := req.SendStreaming()
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	body, err := io.ReadAll(stream)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Status: %d\n", stream.StatusCode())
	fmt.Printf("Body (%d bytes):\n%s\n", len(body), body)
}
