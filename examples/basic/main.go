package main

import (
	"fmt"
	"log"

	"github.com/EZXLabs/lkrequest-go/lkrequest"
)

func main() {
	resp, err := lkrequest.Get("https://httpbin.org/get")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode())
	fmt.Printf("Body:\n%s\n", resp.String())
}
