package test

import (
	"go.uber.org/fx"
)

// LifecycleRecorder captures lifecycle hooks appended during tests.
type LifecycleRecorder struct {
	Hooks []fx.Hook
}

// Append stores hook for later invocation.
func (l *LifecycleRecorder) Append(h fx.Hook) {
	l.Hooks = append(l.Hooks, h)
}

// ShutdownerStub records shutdown invocations.
type ShutdownerStub struct {
	Called chan struct{}
}

// Shutdown notifies tests about graceful termination.
func (s *ShutdownerStub) Shutdown(...fx.ShutdownOption) error {
	if s.Called != nil {
		select {
		case s.Called <- struct{}{}:
		default:
		}
	}
	return nil
}
