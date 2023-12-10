package pending_instances_holder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/horockey/consul_instance_manager/internal/model"
	"github.com/horockey/go-scheduler"
)

type PendingInstancesHolder struct {
	holdPeriod time.Duration
	sched      *scheduler.Scheduler[model.InstanceChange]

	out chan model.InstanceChange
}

func New(holdPeriod time.Duration) (*PendingInstancesHolder, error) {
	sched, err := scheduler.NewScheduler[model.InstanceChange]()
	if err != nil {
		return nil, fmt.Errorf("creating internal scheduler: %w", err)
	}

	return &PendingInstancesHolder{
		holdPeriod: holdPeriod,
		sched:      sched,
		out:        make(chan model.InstanceChange, 100),
	}, nil
}

func (pih *PendingInstancesHolder) Start(ctx context.Context) error {
	errs := make(chan error)
	defer close(errs)

	go func() {
		if err := pih.sched.Start(ctx); err != nil {
			errs <- err
		}
	}()

	var resErr error
	func() {
		for {
			select {
			case ev := <-pih.sched.EmitChan():
				pih.out <- ev.Payload
			case err := <-errs:
				resErr = fmt.Errorf("running internal scheduler: %w", err)
				return
			case <-ctx.Done():
				resErr = ctx.Err()
				return
			}
		}
	}()

	if !errors.Is(resErr, context.Canceled) {
		return fmt.Errorf("running context: %w", resErr)
	}

	return nil
}

func (pih *PendingInstancesHolder) Out() chan model.InstanceChange {
	return pih.out
}

func (pih *PendingInstancesHolder) Add(ins model.Instance) error {
	_, err := pih.sched.Schedule(
		model.InstanceChange{
			Instance: ins,
			IsDown:   true,
		},
		scheduler.After[model.InstanceChange](pih.holdPeriod),
		scheduler.Tag[model.InstanceChange](ins.Name),
	)
	if err != nil {
		return fmt.Errorf("scheduling downed node: %w", err)
	}

	return nil
}

func (pih *PendingInstancesHolder) Remove(ins model.Instance) error {
	err := pih.sched.UnscheduleByTag(ins.Name)
	if err != nil {
		return fmt.Errorf("unscheduling downed node: %w", err)
	}

	return nil
}
