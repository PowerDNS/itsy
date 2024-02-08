package itsy

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// Options for the itsy microservice
type Options struct {
	Config         Config
	Logger         *slog.Logger
	VersionSemVer  string // Must be SemVer compliant, if set. Consider "0.0.0+foo" for other versions.
	VersionFull    string // Can be any arbitrary string
	Name           string // Service name (required)
	Description    string // Service description
	ConnectOptions []nats.Option
}

func New(opt Options) (*Service, error) {
	if opt.Name == "" {
		return nil, errors.New("name option is required")
	}
	if opt.Logger == nil {
		opt.Logger = slog.Default().With("component", "itsy")
	}
	opt.Config = opt.Config.AddEnviron()
	svc := &Service{
		opt:      opt,
		conf:     opt.Config,
		l:        opt.Logger,
		handlers: make(map[string]HandlerFunc),
		Ready:    make(chan struct{}),
	}
	return svc, nil
}

// HandlerFunc is a handler function. It differs from micro.Handler.
type HandlerFunc func(req Request) error

// Service is the main object that describes the microservice
type Service struct {
	opt      Options
	conf     Config
	l        *slog.Logger
	handlers map[string]HandlerFunc

	// Ready is closed when the service is up and running, which can
	// be useful for tests
	Ready chan struct{}
}

// AddHandler registers a HandlerFunc.
// This can only be called before calling Run.
func (s *Service) AddHandler(name string, handler HandlerFunc) {
	s.handlers[name] = handler
}

// Run runs the NATS goroutine
// The natsURL is the connection URL as understood by NATS
// The topo is an optional list of dotted string topologies like "eu.amsterdam.az1"
// Run must only be called once!
func (s *Service) Run(ctx context.Context) error {
	// Topologies determine under which topics we register the service endpoints
	topo := slices.Clone(s.conf.Topologies)
	sort.Strings(topo)
	topoString := strings.Join(topo, " ")

	// Create a NATS connection. This will automatically reconnect when needed.
	connectOpts := []nats.Option{
		nats.RetryOnFailedConnect(true),
		nats.ConnectHandler(func(conn *nats.Conn) {
			s.l.Info("NATS initial connection successful")
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			s.l.Info("NATS reconnected")
		}),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			s.l.Warn("NATS disconnected", "err", err)
		}),
		nats.MaxReconnects(-1), // never give up, defaults to 60
	}
	connectOpts = append(connectOpts, s.opt.ConnectOptions...)
	nc, err := nats.Connect(
		s.conf.URL,
		connectOpts...,
	)
	if err != nil {
		return err
	}

	// Prepare some variables we will need later
	hostname, _ := os.Hostname()

	versionSemVer := s.opt.VersionSemVer
	if versionSemVer == "" {
		versionSemVer = "0.0.0"
	}
	versionFull := s.opt.VersionFull
	if versionFull == "" {
		versionFull = versionSemVer
	}

	prefix := s.conf.Prefix
	if prefix == "" {
		prefix = "svc"
	}

	// Metadata that we associate with this service
	meta := make(map[string]string)
	for k, v := range s.conf.Meta {
		meta[k] = v
	}
	if hostname != "" {
		meta["hostname"] = hostname
	}
	meta["topologies"] = topoString
	meta["version_full"] = versionFull

	// Create the mirco.Service that enabled the common discovery (PING, STATS and INFO)
	// endpoints.
	svcConfig := micro.Config{
		Name:        s.opt.Name,
		Version:     versionSemVer,
		Description: s.opt.Description,
		Metadata:    meta,
	}
	svc, err := micro.AddService(nc, svcConfig)
	if err != nil {
		return err
	}
	svcID := svc.Info().ID

	// Our itsy services return a few common headers with every request.
	// We avoid multiple headers with the same name, because not all client libs support that
	defaultHeaders := micro.Headers{
		"Service-ID":       []string{svcID},
		"Service-Name":     []string{svcConfig.Name},
		"Service-Version":  []string{versionFull},
		"Service-Topology": []string{topoString},
	}
	if hostname != "" {
		defaultHeaders["Service-Hostname"] = []string{hostname}
	}
	withDefaultHeaders := micro.WithHeaders(defaultHeaders)
	ropts := []micro.RespondOpt{withDefaultHeaders}

	// Register all handlers under all topology topics
	// TODO: Perhaps only register the top level one this way and use normal
	//       NATS subscriptions to handle the topology endpoints. That would
	//       require reimplementing the Request object, though.
	for name, handler := range s.handlers {
		g := svc.AddGroup(prefix)
		for _, name := range ExpandTopology(name, topo, svcID) {
			s.l.Info("Adding NATS endpoint", "subject", prefix+"."+name.Full)
			q := "q"
			if name.All {
				// Using unique queue groups ensures that all instances respond
				q = "id." + svcID
			}
			err := g.AddEndpoint(
				strings.Replace(name.Full, ".", "-", -1),
				toMicroHandler(handler, ropts),
				micro.WithEndpointSubject(name.Full),
				micro.WithEndpointQueueGroup(q),
				micro.WithEndpointMetadata(map[string]string{"topo": name.Topo}),
			)
			if err != nil {
				return err
			}
		}
	}

	s.l.Info("NATS services registered", "id", svcID, "topologies", topoString)
	close(s.Ready) // Signal that the service is up and running

	defer s.l.Info("NATS services stopped", "id", svcID)

	// Wait until the context is closed, and then cleanly disconnect from NATS
	<-ctx.Done()
	return nc.Drain()
}

// toMicroHandler converts our HandlerFunc to a micro.HandlerFunc
func toMicroHandler(handler HandlerFunc, ropts []micro.RespondOpt) micro.HandlerFunc {
	return func(mr micro.Request) {
		req := Request{
			mr:   mr,
			opts: ropts,
		}
		if err := handler(req); err != nil {
			_ = req.RespondErr(err)
		}
	}
}
