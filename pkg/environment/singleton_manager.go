package environment

import (
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type singletonManager struct {
	environment Environment
	lock        sync.Mutex
	acquired    bool
}

// NewSingletonManager is a simple Manager that always returns the same
// Environment. This is typically used in combination with
// NewLocalExecutionEnvironment or NewRemoteExecutionManager to force
// that all build actions are executed using the same method.
func NewSingletonManager(environment Environment) Manager {
	em := &singletonManager{}
	em.environment = &singletonEnvironment{
		Environment: environment,
		manager:     em,
	}
	return em
}

func (em *singletonManager) Acquire(actionDigest *util.Digest, platformProperties map[string]string) (Environment, error) {
	em.lock.Lock()
	defer em.lock.Unlock()
	if em.acquired {
		return nil, status.Error(codes.Unavailable, "Environment is already acquired")
	}
	em.acquired = true
	return em.environment, nil
}

type singletonEnvironment struct {
	Environment
	manager *singletonManager
}

func (e *singletonEnvironment) Release() {
	// Never call Release() on the underlying environment, as it
	// will be reused.
	e.manager.lock.Lock()
	e.manager.acquired = false
	e.manager.lock.Unlock()
}
