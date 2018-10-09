package queue

import (
	"net/url"

	"gopkg.in/src-d/go-errors.v0"
)

var (
	// ErrUnsupportedProtocol is the error returned when a Broker does not know
	// how to connect to a given URI.
	ErrUnsupportedProtocol = errors.NewKind("unsupported protocol: %s")
	// ErrMalformedURI is the error returned when a Broker does not know
	// how to parse a given URI.
	ErrMalformedURI = errors.NewKind("malformed connection URI: %s")

	register = make(map[string]BrokerBuilder, 0)
)

// BrokerBuilder instantiates a new Broker based on the given uri.
type BrokerBuilder func(uri string) (Broker, error)

// Register registers a new BrokerBuilder to be used by NewBroker, this function
// should be used in an init function in the implementation packages such as
// `amqp` and `memory`.
func Register(name string, b BrokerBuilder) {
	register[name] = b
}

// NewBroker creates a new Broker based on the given URI. In order to register
// different implementations the package should be imported, example:
//
// 	import _ "gopkg.in/src-d/go-queue.v1/amqp"
func NewBroker(uri string) (Broker, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, ErrMalformedURI.Wrap(err, uri)
	}

	if url.Scheme == "" {
		return nil, ErrMalformedURI.New(uri)
	}

	b, ok := register[url.Scheme]
	if !ok {
		return nil, ErrUnsupportedProtocol.New(url.Scheme)
	}

	return b(uri)
}
