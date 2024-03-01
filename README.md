# Itsy service wrapper for NATS micro

Itsy makes it easy to create [NATS micro](https://pkg.go.dev/github.com/nats-io/nats.go/micro)
services in Go code, with some boilerplate additions we use for our services.

Improvements over the base NATS micro interface:

- Configuration struct that can be included in your own YAML/JSON config.
- Configuration overrides through `ITSY_*` environment variables.
- Handlers can return a Go error, which will be returned to the client.
- Services subscribe to various topology subjects for easy targetting and debugging (see below).
- Responses include various standard headers with information about the responding instance.
- Easy ways to extend the service metadata through configuration or environment variables.
- Sensible connection defaults for services, e.g. keep trying to reconnect.

## Example service

A simple example service (error handling omitted).

```go
func Run(ctx context.Context) {
	// Simple example configuration
	conf := itsy.Config{
		URL:    "nats://localhost:4222",
		Prefix: "example-services",
		Topologies: []string{
			"eu.nl.ams",  
			// ITSY_TOPO="k8s.<clusterid> foo.bar"
		},
		Meta: map[string]string{
			"foo": "bar",
			// ITSY_META_POD=foo-abc-1234
		},
	}

	s, _ := itsy.Start(itsy.Options{
		Config:        conf,
		VersionSemVer: "0.0.1",
		Name:          "itsy-example",
		Description:   "An example service",
	})

	s.MustAddHandler("echo", func(req itsy.Request) error {
		err := req.Respond(req.Data())
		return err // this will try to send an error response, if not nil
	}, nil)
	
	// Do other things here 
	// e.g. wait for the context to close 
	<-ctx.Done
}

```

## Topologies

Topologies are extra subscription subject suffixes that allow you to target specific instances
for debugging or operational needs.

The example service shown above will listen on the following NATS subjects:

- example-services.itsy-example.echo
- example-services.itsy-example.echo.all
- example-services.itsy-example.echo.all.eu
- example-services.itsy-example.echo.all.eu.nl
- example-services.itsy-example.echo.all.eu.nl.ams
- example-services.itsy-example.echo.any
- example-services.itsy-example.echo.any.eu
- example-services.itsy-example.echo.any.eu.nl
- example-services.itsy-example.echo.any.eu.nl.ams
- example-services.itsy-example.echo.id.VFAOuXPQKaJFU9SlETBVZ8

A service can have multiple dotted topologies and listens on all parts of the hierarchy.

When performing a request on the `all` subjects, all online instances that subscribe to the
topic will respond.

When performing a request on the `any` subjects, `id` subject or the base subject, only one
instance will respond.

For example, with two of these services running, you will see the following:

```
$ nats req example-services.itsy-example.echo.all --replies=2 'hello world'
15:18:07 Sending request on "example-services.itsy-example.echo.all"

15:18:07 Received with rtt 388.864µs
15:18:07 Service-Hostname: foo.local
15:18:07 Service-ID: lnmVhnFr9Ft6iNEgb3bKR9
15:18:07 Service-Name: itsy-example
15:18:07 Service-Topology: eu.nl.ams
15:18:07 Service-Version: 0.0.1

hello world


15:18:07 Received with rtt 425.988µs
15:18:07 Service-Hostname: foo.local
15:18:07 Service-ID: VFAOuXPQKaJFU9SlETBVZ8
15:18:07 Service-Name: itsy-example
15:18:07 Service-Topology: eu.nl.ams
15:18:07 Service-Version: 0.0.1

hello world
```

Also note the extra headers that Itsy automatically adds to all responses.


