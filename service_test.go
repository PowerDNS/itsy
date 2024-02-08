package itsy_test

import (
	"context"
	"fmt"
	"time"

	"github.com/PowerDNS/itsy"
	"github.com/nats-io/nats.go"
)

func ExampleService() {
	// NOTE: This test requires a NATS server to be running on localhost:4222
	// Simply run:   nats-server

	// Simple test configuration
	conf := itsy.Config{
		Prefix: "test-itsy",
		Topologies: []string{
			"eu.nl.ams",
		},
	}
	s, err := itsy.New(itsy.Options{
		Config:        conf,
		Logger:        nil,
		VersionSemVer: "0.0.1",
		Name:          "itsy-example",
		Description:   "An example service",
	})
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return
	}

	s.AddHandler("echo", func(req itsy.Request) error {
		err := req.Respond(req.Data())
		return err // this will try to send an error response, if not nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Connect with a client and send a request after a delay
	go func() {
		nc, err := nats.Connect("nats://localhost:4222")
		if err != nil {
			fmt.Printf("ERROR: %v", err)
			return
		}
		defer nc.Close()

		// Wait until the service is running
		<-s.Running

		msg, err := nc.Request("test-itsy.echo.any.eu.nl.ams", []byte("hello world"), 100*time.Millisecond)
		if err != nil {
			fmt.Printf("ERROR: %v", err)
			return
		}
		fmt.Println("Received:", string(msg.Data))
		cancel()
	}()

	// Run the service
	err = s.Run(ctx)
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return
	}

	fmt.Println("OK")

	// Output:
	// Received: hello world
	// OK
}
