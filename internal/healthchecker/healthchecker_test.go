package healthchecker_test

import (
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

type healthCheckerTestSuite struct {
	suite.Suite

	consul testcontainers.Container
}

// TODO
