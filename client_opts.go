package consul_instance_manager

import (
	"errors"

	consul "github.com/hashicorp/consul/api"
	"github.com/horockey/go-toolbox/options"
	"github.com/rs/zerolog"
)

// Sets custom logger.
// Default is stdout console logger.
func WithLogger(l zerolog.Logger) options.Option[Client] {
	return func(target *Client) error {
		target.logger = l
		return nil
	}
}

// Sets custon consul client to operate with.
// Default is client based on DefaultConfig().
func WithConsulClient(cc *consul.Client) options.Option[Client] {
	return func(target *Client) error {
		if cc == nil {
			return errors.New("got nil consul client")
		}
		target.cl = cc
		return nil
	}
}
