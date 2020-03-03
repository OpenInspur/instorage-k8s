package host

import "k8s.io/utils/exec"

func Run(cmd string, args ...string) ([]byte, error) {
	exe := exec.New()
	return exe.Command(cmd, args...).CombinedOutput()
}

// NewFakeExec returns a new FakeExec
func NewFakeExec(run runHook) *FakeExec {
	return &FakeExec{runHook: run}
}

// FakeExec for testing.
type FakeExec struct {
	runHook runHook
}
type runHook func(cmd string, args ...string) ([]byte, error)

// Run executes the command using the optional runhook, if given
func (f *FakeExec) Run(cmd string, args ...string) ([]byte, error) {
	if f.runHook != nil {
		return f.runHook(cmd, args...)
	}
	return nil, nil
}
