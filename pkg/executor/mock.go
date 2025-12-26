package executor

import "context"

// MockExecutor is a mock implementation of Executor for testing.
type MockExecutor struct {
	RunFunc            func(ctx context.Context, name string, args ...string) error
	OutputFunc         func(ctx context.Context, name string, args ...string) ([]byte, error)
	CombinedOutputFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func (m *MockExecutor) Run(ctx context.Context, name string, args ...string) error {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, name, args...)
	}
	return nil
}

func (m *MockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.OutputFunc != nil {
		return m.OutputFunc(ctx, name, args...)
	}
	return []byte{}, nil
}

func (m *MockExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.CombinedOutputFunc != nil {
		return m.CombinedOutputFunc(ctx, name, args...)
	}
	return []byte{}, nil
}
