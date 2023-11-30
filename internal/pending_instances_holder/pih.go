package pending_instances_holder

import (
	"context"
	"sync"
	"time"

	"github.com/horockey/consul_instance_manager/internal/model"
)

type pendingInstancesHolder struct {
	mu sync.RWMutex

	storage    map[model.Instance]time.Time
	holdPeriod time.Duration

	out chan model.InstanceChange
}

func New(holdPeriod time.Duration) *pendingInstancesHolder {
	return &pendingInstancesHolder{
		holdPeriod: holdPeriod,
		storage:    map[model.Instance]time.Time{},
	}
}

func (pih *pendingInstancesHolder) Start(context.Context) error {
	// TODO
	return nil
}

func (pih *pendingInstancesHolder) Out() chan model.InstanceChange {
	return pih.out
}

func (pih *pendingInstancesHolder) Add(ins model.Instance) {
	// TODO
}

func (pih *pendingInstancesHolder) Remove(ins model.Instance) {
	// TODO
}
