package redis_test

import (
	"testing"

	"github.com/nielskrijger/goboot"
	"github.com/nielskrijger/goboot/redis"
	"github.com/stretchr/testify/assert"
)

func TestService_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	s := &redis.Service{}
	assert.Nil(t, s.Configure(goboot.NewAppContext("./testdata", "valid")))
	assert.Nil(t, s.Init())
	assert.Equal(t, "Redis<0.0.0.0:6379 db:3>", s.Client.String())
}

func TestService_ErrorMissingConfig(t *testing.T) {
	s := &redis.Service{}
	err := s.Configure(goboot.NewAppContext("./testdata", ""))
	assert.EqualError(t, err, "missing redis configuration")
}

func TestService_ErrorEmptyURL(t *testing.T) {
	s := &redis.Service{}
	err := s.Configure(goboot.NewAppContext("./testdata", "no-url"))
	assert.EqualError(t, err, "config \"redis.url\" is required")
}

func TestService_ErrorOnConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	s := &redis.Service{}
	err := s.Configure(goboot.NewAppContext("./testdata", "invalid"))
	assert.EqualError(t, err, "failed to connect to redis after 5 retries: dial tcp 1.2.3.4:6379: i/o timeout")
}
