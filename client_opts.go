package go-consul-instance-manager

import (
	"errors"
	"fmt"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/horockey/go-toolbox/options"
	"github.com/rs/zerolog"
	"github.com/serialx/hashring"
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

// Sets duration, for which node will be marked as pending before being removed from instances list.
// Default is 15s.
func WithDownHoldDuration(dur time.Duration) options.Option[Client] {
	return func(target *Client) error {
		if dur <= 0 {
			return fmt.Errorf("duration must be positive, got: %d", dur)
		}

		target.holdDur = dur
		return nil
	}
}

// Sets interval to check instances list.
// Default is 1s.
func WithPollInterval(dur time.Duration) options.Option[Client] {
	return func(target *Client) error {
		if dur <= 0 {
			return fmt.Errorf("duration must be positive, got: %d", dur)
		}

		target.pollInterval = dur
		return nil
	}
}

func WithBackupHashring(hashFunc hashring.HashFunc) options.Option[Client] {
	return func(target *Client) error {
		if hashFunc == nil {
			return errors.New("got nil backup hashfunc")
		}

		target.hashrings = append(target.hashrings, hashring.NewWithHash([]string{}, hashFunc))
		return nil
	}
}
