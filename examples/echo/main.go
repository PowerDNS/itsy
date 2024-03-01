package main

import (
	"context"

	"github.com/PowerDNS/itsy"
)

func Run(ctx context.Context) error {
	// Simple example configuration
	conf := itsy.Config{
		URL:    "nats://localhost:4222",
		Prefix: "example-services",
		Topologies: []string{
			"eu.nl.ams",
		},
		Meta: map[string]string{
			"foo": "bar",
		},
	}

	s, err := itsy.Start(itsy.Options{
		Config:        conf,
		VersionSemVer: "0.0.1",
		Name:          "itsy-example",
		Description:   "An example service",
	})
	if err != nil {
		return err
	}
	defer s.Stop()

	s.MustAddHandler("echo", func(req itsy.Request) error {
		err := req.Respond(req.Data())
		return err // this will try to send an error response, if not nil
	}, nil)

	// Run the service (blocks until it exits)
	<-ctx.Done()
	return nil
}

func main() {
	err := Run(context.Background())
	if err != nil {
		panic(err)
	}
}
