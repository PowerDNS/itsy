package itsy_test

import (
	"context"
	"fmt"
	"time"

	"github.com/PowerDNS/itsy"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func ExampleService() {
	// Start an in-process NATS server for this example
	server, err := natsserver.NewServer(&natsserver.Options{
		DontListen: true,
	})
	check(err)
	server.Start()
	defer server.Shutdown()
	if !server.ReadyForConnections(time.Second * 5) {
		panic("NATS server didn't start")
	}

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
		ConnectOptions: []nats.Option{
			nats.InProcessServer(server),
		},
	})
	check(err)

	s.AddHandler("echo", func(req itsy.Request) error {
		err := req.Respond(req.Data())
		return err // this will try to send an error response, if not nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Connect with a client and send a request when ready
	go func() {
		nc, err := nats.Connect("", nats.InProcessServer(server))
		check(err)
		defer nc.Close()

		// Wait until the service is running
		<-s.Ready

		msg, err := nc.Request("test-itsy.echo.any.eu.nl.ams", []byte("hello world"), time.Second)
		check(err)
		fmt.Println("Received:", string(msg.Data))
		cancel()
	}()

	// Run the service
	err = s.Run(ctx)
	check(err)

	fmt.Println("OK")

	// Output:
	// Received: hello world
	// OK
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
