package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alarmistdev/status/check"
	dockertypes "github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
)

type stubDockerClient struct {
	containers []dockertypes.Container
	err        error
}

func (s *stubDockerClient) ContainerList(
	_ context.Context,
	_ containerTypes.ListOptions,
) ([]dockertypes.Container, error) {
	if s.err != nil {
		return nil, s.err
	}

	return append([]dockertypes.Container(nil), s.containers...), nil
}

func TestCheck_HealthyContainers(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"app": "status", "env": "prod"}
	client := &stubDockerClient{
		containers: []dockertypes.Container{
			{ID: "abc", Labels: labels, State: "running"},
			{ID: "def", Labels: labels, State: "running"},
		},
	}

	checker, err := newCheck(labels, 20*time.Millisecond, check.Config{Timeout: 50 * time.Millisecond}, client)
	if err != nil {
		t.Fatalf("unexpected error creating check: %v", err)
	}
	defer checker.Close()

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expected healthy check, got error: %v", err)
	}
}

func TestCheck_FailsWhenNoContainersMatch(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"app": "status"}
	client := &stubDockerClient{}

	checker, err := newCheck(labels, 20*time.Millisecond, check.Config{Timeout: 50 * time.Millisecond}, client)
	if err != nil {
		t.Fatalf("unexpected error creating check: %v", err)
	}
	defer checker.Close()

	if err := checker.Check(context.Background()); err == nil {
		t.Fatalf("expected error when no containers match labels")
	}
}

func TestCheck_DetectsContainersStopping(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"app": "status"}
	client := &stubDockerClient{
		containers: []dockertypes.Container{
			{ID: "abc", Labels: labels, State: "running"},
		},
	}

	checker, err := newCheck(labels, 15*time.Millisecond, check.Config{Timeout: 50 * time.Millisecond}, client)
	if err != nil {
		t.Fatalf("unexpected error creating check: %v", err)
	}
	defer checker.Close()

	waitForHealthy(t, checker)

	client.containers = []dockertypes.Container{
		{ID: "abc", Labels: labels, State: "exited"},
	}

	waitForFailure(t, checker)
}

func TestCheck_PropagatesDockerErrors(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"app": "status"}
	client := &stubDockerClient{
		err: errors.New("docker unreachable"),
	}

	checker, err := newCheck(labels, 20*time.Millisecond, check.Config{Timeout: 50 * time.Millisecond}, client)
	if err != nil {
		t.Fatalf("unexpected error creating check: %v", err)
	}
	defer checker.Close()

	if err := checker.Check(context.Background()); err == nil {
		t.Fatalf("expected docker error to propagate")
	}
}

func waitForHealthy(t *testing.T, checker check.Check) {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := checker.Check(context.Background()); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("checker did not become healthy before deadline")
}

func waitForFailure(t *testing.T, checker check.Check) {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := checker.Check(context.Background()); err != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("checker did not report failure before deadline")
}
