package go_consul_instance_manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/serialx/hashring"

	consul_iman "github.com/horockey/go-consul-instance-manager"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

const (
	serviceName = "test_service"
	hostName1   = "host1"
	hostName2   = "host2"
	addr1       = "http://host1:8080"
	addr2       = "http://host2:8080"
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

	err := s.iman.Register(hostName1, addr1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err := s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName1, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())
}

func (s *ImanTestSuite) TestIman_InstanceDown_AndRecover() {
	ctx := context.TODO()
	t := s.T()

	go s.iman.Start(ctx)

	err := s.iman.Register(hostName1, addr1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err := s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName1, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())

	err = s.iman.Deregister(hostName1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err = s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName1, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusPending, inses[0].Status())

	err = s.iman.Register(hostName1, addr1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err = s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName1, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())
}

func (s *ImanTestSuite) TestIman_InstanceDown_Finally() {
	ctx := context.TODO()
	t := s.T()

	go s.iman.Start(ctx)

	err := s.iman.Register(hostName1, addr1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 600)

	inses, err := s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 1)
	require.Equal(t, hostName1, inses[0].Name())
	require.Equal(t, consul_iman.InstanceStatusAlive, inses[0].Status())

	err = s.iman.Deregister(hostName1)
	require.NoError(t, err)
	time.Sleep(time.Second * 2)

	inses, err = s.iman.GetInstances()
	require.NoError(t, err)
	require.Len(t, inses, 0)
}

func (s *ImanTestSuite) TestMultipleHashrings() {
	t := s.T()
	ctx := context.TODO()

	consulCfg := api.DefaultConfig()
	consulCfg.Address = consulAddr
	consulClient, err := api.NewClient(consulCfg)
	require.NoError(t, err)

	iman, err := consul_iman.NewClient(
		serviceName,
		consul_iman.WithDownHoldDuration(time.Second),
		consul_iman.WithPollInterval(time.Millisecond*500),
		consul_iman.WithConsulClient(consulClient),
		consul_iman.WithBackupHashring(func(b []byte) hashring.HashKey {
			return hashKey(string(b))
		}),
	)
	require.NoError(t, err)

	go iman.Start(ctx)

	err = s.iman.Register(hostName1, addr1)
	require.NoError(t, err)
	err = s.iman.Register(hostName2, addr1)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 1000)

	inses, err := iman.GetDataHolders("abc")
	require.NoError(t, err)
	require.Len(t, inses, 2)
	require.Equal(t, inses[0].Name(), hostName1)
	require.Equal(t, inses[1].Name(), hostName2)
}

type hashKey string

func (hk hashKey) Less(other hashring.HashKey) bool {
	return hk < other.(hashKey)
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
