package itsy_test

import (
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
	s, err := itsy.Start(itsy.Options{
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
	defer s.Stop()

	s.MustAddHandler("echo", func(req itsy.Request) error {
		err := req.Respond(req.Data())
		return err // this will try to send an error response, if not nil
	}, nil)

	// Connect with a client and send a request when ready
	nc, err := nats.Connect("", nats.InProcessServer(server))
	check(err)
	defer nc.Close()

	msg, err := nc.Request("test-itsy.echo.any.eu.nl.ams", []byte("hello world"), time.Second)
	check(err)
	fmt.Println("Received:", string(msg.Data))

	fmt.Println("Done")

	// Output:
	// Received: hello world
	// Done
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
