package environment

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

type singletonManager struct {
	environment Environment
}

// NewSingletonManager is a simple Manager that always returns the same
// Environment. This is typically used in combination with
// NewLocalExecutionEnvironment or NewRemoteExecutionManager to force
// that all build actions are executed using the same method.
func NewSingletonManager(environment Environment) Manager {
	return &singletonManager{
		environment: &nonReleasingEnvironment{
			Environment: environment,
		},
	}
}

func (em *singletonManager) Acquire(actionDigest *util.Digest, platformProperties map[string]string) (Environment, error) {
	return em.environment, nil
}

type nonReleasingEnvironment struct {
	Environment
}

func (e *nonReleasingEnvironment) Release() {
	// Never call Release() on the underlying environment, as it
	// will be reused.
}
