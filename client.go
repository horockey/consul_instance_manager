package go_consul_instance_manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	consul "github.com/hashicorp/consul/api"
	"github.com/horockey/go-consul-instance-manager/internal/healthchecker"
	"github.com/horockey/go-consul-instance-manager/internal/pending_instances_holder"
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

	appName   string
	hashrings []*hashring.HashRing

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
		hashrings:     []*hashring.HashRing{hashring.New([]string{})},
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
				for idx, hr := range cl.hashrings {
					cl.hashrings[idx] = hr.AddNode(ev.Instance.Name)
				}
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
			for idx, hr := range cl.hashrings {
				cl.hashrings[idx] = hr.RemoveNode(ev.Instance.Name)
			}
			cl.mu.Unlock()

		case <-ctx.Done():
			resErr = errors.Join(resErr, fmt.Errorf("running context: %w", ctx.Err()))
		}
	}

	return resErr
}

// Get list of alive instances now.
// Client must be started to run this method properly.
func (cl *Client) GetInstances() ([]*Instance, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return maps.Values(cl.instances), nil
}

// Get list of instances that hold given key.
// Client must be started to run this method properly.
func (cl *Client) GetDataHolders(key string) ([]*Instance, error) {
	inses := map[*Instance]struct{}{}

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	for _, hr := range cl.hashrings {
		node, ok := hr.GetNode(key)
		if !ok {
			return nil, fmt.Errorf("data holder for key %s not found", key)
		}

		ins, found := cl.instances[node]
		if !found {
			return nil, fmt.Errorf("unknow instance node: %s", node)
		}

		inses[ins] = struct{}{}
	}

	keys := maps.Keys(inses)
	sort.Slice(keys, func(i, j int) bool { return keys[i].Name() < keys[j].Name() })

	return keys, nil
}

// Registers new instance of cl.appName with given parameters.
func (cl *Client) Register(hostname string, address string) error {
	if _, err := cl.cl.Catalog().Register(&consul.CatalogRegistration{
		ID:      uuid.NewString(),
		Node:    hostname,
		Address: address,
		Service: &consul.AgentService{
			ID:      cl.appName + "_" + hostname,
			Service: cl.appName,
		},
		Checks: consul.HealthChecks{
			{
				Node:    hostname,
				CheckID: uuid.NewString(),
				Status:  consul.HealthPassing,
			},
		},
	}, nil); err != nil {
		return fmt.Errorf("registering in consul: %w", err)
	}

	return nil
}

// Deregisters instance of cl.appName with given parameters.
func (cl *Client) Deregister(hostname string) error {
	_, err := cl.cl.Catalog().Deregister(&consul.CatalogDeregistration{
		Node:      hostname,
		ServiceID: cl.appName + "_" + hostname,
	}, nil)
	if err != nil {
		return fmt.Errorf("deregistering from consul: %w", err)
	}

	return nil
}
