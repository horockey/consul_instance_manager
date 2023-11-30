package consul_instance_manager

import (
	"context"
	"fmt"
	"os"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/horockey/go-toolbox/options"
	"github.com/rs/zerolog"
	"github.com/serialx/hashring"
)

type Comparable interface{ comparable }

type Client struct {
	cl *consul.Client

	appName  string
	hashring *hashring.HashRing

	logger zerolog.Logger
}

// Creates new client to operate with.
func NewClient(appName string, opts ...options.Option[Client]) (*Client, error) {
	cc, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("creating consul client: %w", err)
	}

	client := Client{
		cl:      cc,
		appName: appName,
		logger: zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger(),
	}

	if err := options.ApplyOptions(&client, opts...); err != nil {
		return nil, fmt.Errorf("applying opts: %w", err)
	}

	return &client, nil
}

func (cl *Client) Start(ctx context.Context) error {
	// TODO
	return nil
}

func (cl *Client) GetInstances() ([]*Instance, error) {
	// TODO
	return nil, nil
}

func (cl *Client) GetDataHolder(key string) (*Instance, error) {
	// TODO
	return nil, nil
}
