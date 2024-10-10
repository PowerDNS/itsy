package itsy

import (
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
	Name           string // Service name (required); also used for subject if SubjectName is not set
	SubjectName    string // Name part as used in subject if different from Name
	Description    string // Service description
	ConnectOptions []nats.Option
}

// Start starts a new Itsy NATS Service in the background.
// The returned Service object can be used to register handlers and control the
// instance.
func Start(opt Options) (*Service, error) {
	if opt.Name == "" {
		return nil, errors.New("name option is required")
	}
	if opt.Logger == nil {
		opt.Logger = slog.Default().With("component", "itsy")
	}
	opt.Config = opt.Config.AddEnviron()
	svc := &Service{
		opt:  opt,
		conf: opt.Config,
		l:    opt.Logger,
	}
	err := svc.start()
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// HandlerFunc is a handler function. It differs from micro.Handler.
type HandlerFunc func(req Request) error

// HandlerOptions are options that influence how a handler is registered.
// This struct is currently empty, but allows for future expansion.
type HandlerOptions struct {
	Global bool // Do not scope the subject with the service name
}

// Service is the main object that describes the microservice
type Service struct {
	// Initialization
	opt   Options
	conf  Config
	l     *slog.Logger
	nc    *nats.Conn
	svc   micro.Service
	ropts []micro.RespondOpt
	topo  []string
}

// Conn returns the underlying NATS connection
func (s *Service) Conn() *nats.Conn {
	return s.nc
}

// ID returns the unique service ID of this instance
func (s *Service) ID() string {
	return s.svc.Info().ID
}

// AddHandler registers a HandlerFunc.
// It returns an error if the name or an option is valid.
func (s *Service) AddHandler(name string, handler HandlerFunc, opts *HandlerOptions) error {
	prefix := s.conf.Prefix
	if prefix == "" {
		prefix = "svc"
	}
	globalName := opts != nil && opts.Global
	if !globalName {
		svcName := s.opt.Name
		if s.opt.SubjectName != "" {
			svcName = s.opt.SubjectName
		}
		prefix += "." + svcName
	}
	svcID := s.ID()

	g := s.svc.AddGroup(prefix)
	for _, name := range ExpandTopology(name, s.topo, svcID) {
		s.l.Info("Adding NATS endpoint", "subject", prefix+"."+name.Full)
		q := "q"
		if name.All {
			// Using unique queue groups ensures that all instances respond
			q = "id." + svcID
		}
		err := g.AddEndpoint(
			strings.Replace(name.Full, ".", "-", -1),
			toMicroHandler(handler, s.ropts),
			micro.WithEndpointSubject(name.Full),
			micro.WithEndpointQueueGroup(q),
			micro.WithEndpointMetadata(map[string]string{"topo": name.Topo}),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// MustAddHandler registers a HandlerFunc, and panics if anything goes wrong.
// Nothing will go wrong if the name and options are valid.
func (s *Service) MustAddHandler(name string, handler HandlerFunc, opts *HandlerOptions) {
	if err := s.AddHandler(name, handler, opts); err != nil {
		panic(err)
	}
}

// Stop stops the NATS service. Once stopped, the object cannot be used again.
func (s *Service) Stop() {
	err := s.nc.Drain()
	if err != nil {
		s.l.Warn("error while closing NATS service", "err", err)
	}
}

// start starts the service in the background.
func (s *Service) start() error {
	// Topologies determine under which topics we register the service endpoints
	topo := slices.Clone(s.conf.Topologies)
	sort.Strings(topo)
	s.topo = topo
	topoString := strings.Join(topo, " ")

	// Create a NATS connection. This will automatically reconnect when needed.
	// All of these can be overridden with custom ConnectOptions.
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
		nats.MaxReconnects(-1), // never give up, the default was 60
	}
	connectOpts = append(connectOpts, s.opt.ConnectOptions...)
	nc, err := nats.Connect(
		s.conf.URL,
		connectOpts...,
	)
	if err != nil {
		return err
	}
	s.nc = nc

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
	s.svc = svc
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
	s.ropts = []micro.RespondOpt{withDefaultHeaders}

	s.l.Info("NATS services registered", "id", svcID, "topologies", topoString)
	return nil
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
