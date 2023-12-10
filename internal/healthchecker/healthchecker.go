package healthchecker

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/horockey/consul_instance_manager/internal/model"
	"github.com/rs/zerolog"
)

type HealthChecker struct {
	cl          *consul.Client
	serviceName string

	lastScanAlives []model.Instance
	pollInterval   time.Duration

	out chan model.InstanceChange

	logger zerolog.Logger
}

func New(
	cl *consul.Client,
	serviceName string,
	pollInterval time.Duration,
	outChanSize uint,
	logger zerolog.Logger,
) *HealthChecker {
	return &HealthChecker{
		cl:             cl,
		lastScanAlives: []model.Instance{},
		serviceName:    serviceName,
		pollInterval:   pollInterval,
		out:            make(chan model.InstanceChange, outChanSize),
		logger:         logger,
	}
}

func (hc *HealthChecker) Out() chan model.InstanceChange {
	return hc.out
}

func (hc *HealthChecker) Start(ctx context.Context) error {
	if err := hc.scan(); err != nil {
		hc.logger.Error().
			Err(fmt.Errorf("scanning alive nodes: %w", err)).
			Send()
	}

	ticker := time.NewTicker(hc.pollInterval)
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return fmt.Errorf("running context: %w", ctx.Err())
		case <-ticker.C:
			if err := hc.scan(); err != nil {
				hc.logger.Error().
					Err(fmt.Errorf("scanning alive nodes: %w", err)).
					Send()
			}
		}
	}
}

func (hc *HealthChecker) scan() error {
	entries, _, err := hc.cl.Catalog().Service(hc.serviceName, "", nil)
	if err != nil {
		return fmt.Errorf("getting service entries: %w", err)
	}

	alives := make([]model.Instance, 0, len(entries))

	for _, entry := range entries {
		if entry.Checks.AggregatedStatus() != consul.HealthPassing {
			continue
		}

		alives = append(alives, model.Instance{
			Name:    entry.Node,
			Address: entry.Address,
		})
	}

	upped := slices.DeleteFunc(alives, func(el model.Instance) bool {
		return slices.Contains(hc.lastScanAlives, el)
	})

	downed := slices.DeleteFunc(hc.lastScanAlives, func(el model.Instance) bool {
		return slices.Contains(alives, el)
	})

	for _, ins := range upped {
		hc.out <- model.InstanceChange{
			Instance: ins,
			IsDown:   false,
		}
	}

	for _, ins := range downed {
		hc.out <- model.InstanceChange{
			Instance: ins,
			IsDown:   true,
		}
	}

	return nil
}
