package consul_instance_manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/horockey/consul_instance_manager/internal/healthchecker"
	"github.com/horockey/consul_instance_manager/internal/pending_instances_holder"
	"github.com/horockey/go-toolbox/options"
	"github.com/rs/zerolog"
	"github.com/serialx/hashring"
	"golang.org/x/exp/maps"
)

type Comparable interface{ comparable }

type Client struct {
	mu        sync.RWMutex
	instances map[string]*Instance

	cl *consul.Client

	appName  string
	hashring *hashring.HashRing

	pih     *pending_instances_holder.PendingInstancesHolder
	holdDur time.Duration

	healthChecker *healthchecker.HealthChecker
	pollInterval  time.Duration
	hcOutChanSize uint

	logger zerolog.Logger
}

// Creates new client to operate with.
func NewClient(appName string, opts ...options.Option[Client]) (*Client, error) {
	cc, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("creating consul client: %w", err)
	}

	client := Client{
		cl:            cc,
		appName:       appName,
		instances:     map[string]*Instance{},
		holdDur:       time.Second * 15,
		pollInterval:  time.Second,
		hcOutChanSize: 100,
		hashring:      hashring.New([]string{}),
		logger: zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger(),
	}

	if err := options.ApplyOptions(&client, opts...); err != nil {
		return nil, fmt.Errorf("applying opts: %w", err)
	}

	client.pih, err = pending_instances_holder.New(client.holdDur)
	if err != nil {
		return nil, fmt.Errorf("creating PIH: %w", err)
	}

	client.healthChecker = healthchecker.New(
		client.cl,
		client.appName,
		client.pollInterval,
		client.hcOutChanSize,
		client.logger,
	)

	return &client, nil
}

func (cl *Client) Start(ctx context.Context) (resErr error) {
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cl.healthChecker.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			resErr = errors.Join(resErr, fmt.Errorf("running healthchecker: %w", err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cl.pih.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			resErr = errors.Join(resErr, fmt.Errorf("running PIH: %w", err))
		}
	}()

	for resErr == nil {
		select {
		case ev := <-cl.healthChecker.Out():
			switch {
			default:
				cl.mu.Lock()
				cl.instances[ev.Instance.Name] = &Instance{
					name:    ev.Instance.Name,
					address: ev.Instance.Address,
					status:  InstanceStatusAlive,
				}
				cl.hashring = cl.hashring.AddNode(ev.Instance.Name)
				cl.mu.Unlock()
			case ev.IsDown:
				cl.mu.Lock()
				cl.instances[ev.Instance.Name] = &Instance{
					name:    ev.Instance.Name,
					address: ev.Instance.Address,
					status:  InstanceStatusPending,
				}
				cl.mu.Unlock()

				if err := cl.pih.Add(ev.Instance); err != nil {
					cl.logger.Error().
						Err(fmt.Errorf("adding instance to PIH: %w", err)).
						Send()
				}
			}

		case ev := <-cl.pih.Out():
			cl.mu.Lock()
			delete(cl.instances, ev.Instance.Name)
			cl.hashring = cl.hashring.RemoveNode(ev.Instance.Name)
			cl.mu.Unlock()

		case <-ctx.Done():
			resErr = errors.Join(resErr, fmt.Errorf("running context: %w", ctx.Err()))
		}
	}

	return resErr
}

func (cl *Client) GetInstances() ([]*Instance, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return maps.Values(cl.instances), nil
}

func (cl *Client) GetDataHolder(key string) (*Instance, error) {
	node, ok := cl.hashring.GetNode(key)
	if !ok {
		return nil, fmt.Errorf("data holder for key %s not found", key)
	}

	cl.mu.RLock()
	ins, found := cl.instances[node]
	cl.mu.RUnlock()
	if !found {
		return nil, fmt.Errorf("unknow instance node: %s", node)
	}

	return ins, nil
}
