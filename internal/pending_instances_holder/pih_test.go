package pending_instances_holder_test

import (
	"context"
	"testing"
	"time"

	"github.com/horockey/consul_instance_manager/internal/model"
	"github.com/horockey/consul_instance_manager/internal/pending_instances_holder"
	"github.com/stretchr/testify/require"
)

var instance = model.Instance{
	Name:    "node1",
	Address: "localhost:8081",
}

func TestAdd(t *testing.T) {
	pihDur := time.Second
	pih, err := pending_instances_holder.New(pihDur)
	require.NoError(t, err)

	go func() {
		err := pih.Start(context.TODO())
		require.NoError(t, err)
	}()

	ts := time.Now()
	err = pih.Add(instance)
	require.NoError(t, err)

	ev := <-pih.Out()
	require.Equal(t, instance, ev.Instance)
	require.True(t, ev.IsDown)
	require.WithinDuration(t, ts.Add(pihDur), time.Now(), time.Millisecond*50)
}

func TestRemove(t *testing.T) {
	pihDur := time.Second
	pih, err := pending_instances_holder.New(pihDur)
	require.NoError(t, err)

	go func() {
		err := pih.Start(context.TODO())
		require.NoError(t, err)
	}()

	err = pih.Add(instance)
	require.NoError(t, err)

	err = pih.Remove(instance)
	require.NoError(t, err)

	timer := time.NewTimer(time.Millisecond * 1500)
	defer timer.Stop()

	select {
	case ev := <-pih.Out():
		t.Fatalf("unexpected emission of event: %+v", ev)
	case <-timer.C:
		return
	}
}
