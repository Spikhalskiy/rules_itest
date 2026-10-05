package runner

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"rules_itest/svclib"
)

func TestWaitUntilHealthyTaskErrorIncludesLabel(t *testing.T) {
	wantErr := errors.New("task failed")
	service := &ServiceInstance{
		VersionedServiceSpec: svclib.VersionedServiceSpec{ServiceSpec: svclib.ServiceSpec{
			Type:  "task",
			Label: "//example:setup",
		}},
		waitErrFn: func() error { return wantErr },
	}

	err := service.WaitUntilHealthy(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("WaitUntilHealthy() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), service.Label) {
		t.Fatalf("WaitUntilHealthy() error = %q, want service label %q", err, service.Label)
	}
}

func TestWaitUntilHealthyServiceErrorIncludesLabel(t *testing.T) {
	wantErr := errors.New("service failed")
	service := &ServiceInstance{
		VersionedServiceSpec: svclib.VersionedServiceSpec{ServiceSpec: svclib.ServiceSpec{
			Type:                "service",
			Label:               "//example:server",
			HealthCheckInterval: "1ms",
		}},
		runErr: wantErr,
	}

	err := service.WaitUntilHealthy(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("WaitUntilHealthy() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), service.Label) {
		t.Fatalf("WaitUntilHealthy() error = %q, want service label %q", err, service.Label)
	}
}

func TestWaitUntilHealthyContextErrorIncludesLabel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &ServiceInstance{
		VersionedServiceSpec: svclib.VersionedServiceSpec{ServiceSpec: svclib.ServiceSpec{
			Type:                "service",
			Label:               "//example:server",
			HealthCheckInterval: "1ms",
		}},
	}

	err := service.WaitUntilHealthy(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WaitUntilHealthy() error = %v, want wrapped context cancellation", err)
	}
	if !strings.Contains(err.Error(), "never became healthy") {
		t.Fatalf("WaitUntilHealthy() error = %q, want \"never became healthy\" classification", err)
	}
	if strings.Contains(err.Error(), "exited") {
		t.Fatalf("WaitUntilHealthy() error = %q, context expiry must not be reported as a process exit", err)
	}
	if !strings.Contains(err.Error(), service.Label) {
		t.Fatalf("WaitUntilHealthy() error = %q, want service label %q", err, service.Label)
	}
}

func TestWaitUntilHealthyDoneBeforeRunErrRecordedWrapsExitError(t *testing.T) {
	cmd := exec.Command("false")
	waitErr := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		t.Fatalf("cmd.Run() error = %v, want *exec.ExitError", waitErr)
	}
	service := &ServiceInstance{
		VersionedServiceSpec: svclib.VersionedServiceSpec{ServiceSpec: svclib.ServiceSpec{
			Type:                "service",
			Label:               "//example:server",
			HealthCheckInterval: "1ms",
		}},
		cmd:       cmd,
		waitErrFn: func() error { return waitErr },
		done:      true,
	}

	err := service.WaitUntilHealthy(context.Background())
	if !errors.As(err, &exitErr) {
		t.Fatalf("WaitUntilHealthy() error = %v, want wrapped *exec.ExitError", err)
	}
	if !strings.Contains(err.Error(), service.Label) || !strings.Contains(err.Error(), "exited before becoming healthy") {
		t.Fatalf("WaitUntilHealthy() error = %q, want label and exit classification", err)
	}
}
