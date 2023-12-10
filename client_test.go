package consul_instance_manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	consul_iman "github.com/horockey/consul_instance_manager"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

const (
	serviceName = "test_service"
	hostName    = "host1"
	addr        = "http://host1:8080"
	consulAddr  = "localhost:8500"
)

type ImanTestSuite struct {
	suite.Suite

	iman   *consul_iman.Client
	consul testcontainers.Container
}

func (s *ImanTestSuite) SetupTest() {
	t := s.T()
	ctx := context.TODO()

	consulCfg := api.DefaultConfig()
	consulCfg.Address = consulAddr
	consulClient, err := api.NewClient(consulCfg)
	require.NoError(t, err)

	s.iman, err = consul_iman.NewClient(
		serviceName,
		consul_iman.WithDownHoldDuration(time.Second),
		consul_iman.WithPollInterval(time.Millisecond*500),
		consul_iman.WithConsulClient(consulClient),
	)
	require.NoError(t, err)

	s.consul, err = setupConsul()
	require.NoError(t, err)

	err = s.consul.Start(ctx)
	require.NoError(t, err)
	time.Sleep(time.Second)
}

func (s *ImanTestSuite) TearDownTest() {
	err := s.consul.Terminate(context.TODO())
	require.NoError(s.T(), err)
}

func TestImanTestSuite(t *testing.T) {
	suite.Run(t, &ImanTestSuite{})
}

func (s *ImanTestSuite) TestIman_Normal() {
	ctx := context.TODO()
	t := s.T()

	go s.iman.Start(ctx)

	err := s.iman.Register(hostName, addr)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err := s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())
}

func (s *ImanTestSuite) TestIman_InstanceDown_AndRecover() {
	ctx := context.TODO()
	t := s.T()

	go s.iman.Start(ctx)

	err := s.iman.Register(hostName, addr)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err := s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())

	err = s.iman.Deregister(hostName)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err = s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusPending, inses[0].Status())

	err = s.iman.Register(hostName, addr)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err = s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())
}

func (s *ImanTestSuite) TestIman_InstanceDown_Finally() {
	ctx := context.TODO()
	t := s.T()

	go s.iman.Start(ctx)

	err := s.iman.Register(hostName, addr)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err := s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())

	err = s.iman.Deregister(hostName)
	require.NoError(t, err)
	time.Sleep(time.Second * 2)

	inses, err = s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 0)
}

func setupConsul() (testcontainers.Container, error) {
	return testcontainers.GenericContainer(context.TODO(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "hashicorp/consul:latest",
			ExposedPorts: []string{"8500:8500"},
			Name:         "consul",
			Hostname:     "consul",
			Networks:     []string{"testnet"},
		},
	})
}
