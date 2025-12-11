package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/alarmistdev/status/check"
	dockertypes "github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type dockerClient interface {
	ContainerList(ctx context.Context, options containerTypes.ListOptions) ([]dockertypes.Container, error)
}

// dockerCheck periodically verifies that containers matching the provided labels are running.
type dockerCheck struct {
	client   dockerClient
	labels   map[string]string
	interval time.Duration
	timeout  time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu          sync.RWMutex
	lastErr     error
	lastChecked time.Time
}

// Check creates a health check that periodically ensures all containers with the given labels are running.
// The check uses the local Docker socket and verifies that every matching container is in the "running" state.
// It fails if no containers match, if any matched container is not running, or if the background result is stale.
func Check(labels map[string]string, interval time.Duration, config check.Config) (check.Check, error) {
	return newCheck(labels, interval, config, nil)
}

func newCheck(
	labels map[string]string,
	interval time.Duration,
	config check.Config,
	cli dockerClient,
) (*dockerCheck, error) {
	if len(labels) == 0 {
		return nil, errors.New("labels must not be empty")
	}
	if interval <= 0 {
		return nil, errors.New("interval must be positive")
	}
	if config.Timeout == 0 {
		config.Timeout = check.DefaultConfig().Timeout
	}

	var err error

	if cli == nil {
		cli, err = client.NewClientWithOpts(
			client.WithHost(client.DefaultDockerHost),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return nil, fmt.Errorf("create docker client: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	dc := &dockerCheck{
		client:   cli,
		labels:   labels,
		interval: interval,
		timeout:  config.Timeout,
		ctx:      ctx,
		cancel:   cancel,
	}

	dc.refresh(ctx)

	dc.wg.Add(1)
	go dc.loop()

	return dc, nil
}

func (dc *dockerCheck) loop() {
	defer dc.wg.Done()

	ticker := time.NewTicker(dc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.refresh(dc.ctx)
		}
	}
}

func (dc *dockerCheck) refresh(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, dc.timeout)
	defer cancel()

	err := dc.checkContainers(ctx)

	dc.mu.Lock()
	dc.lastErr = err
	dc.lastChecked = time.Now()
	dc.mu.Unlock()
}

func (dc *dockerCheck) checkContainers(ctx context.Context) error {
	args := filters.NewArgs()
	for k, v := range dc.labels {
		args.Add("label", fmt.Sprintf("%s=%s", k, v))
	}

	containers, err := dc.client.ContainerList(ctx, containerTypes.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return fmt.Errorf("list docker containers: %w", err)
	}

	matched := 0
	for _, container := range containers {
		if !dc.matchesLabels(container.Labels) {
			continue
		}
		matched++

		if container.State != "running" {
			return fmt.Errorf("container %s not running (state=%s)", container.ID, container.State)
		}
	}

	if matched == 0 {
		return fmt.Errorf("no containers found with labels %v", dc.labels)
	}

	return nil
}

func (dc *dockerCheck) matchesLabels(found map[string]string) bool {
	for key, expected := range dc.labels {
		if found[key] != expected {
			return false
		}
	}

	return true
}

// Check implements the check.Check interface by returning the most recent background result.
func (dc *dockerCheck) Check(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("docker check context: %w", ctx.Err())
	default:
	}

	dc.mu.RLock()
	lastErr := dc.lastErr
	lastChecked := dc.lastChecked
	dc.mu.RUnlock()

	if lastChecked.IsZero() {
		return errors.New("docker check has not completed an initial run")
	}

	if time.Since(lastChecked) > dc.interval*2 {
		return fmt.Errorf("docker check result stale: last=%s interval=%s", time.Since(lastChecked), dc.interval)
	}

	return lastErr
}

// Close stops the background loop and closes the Docker client when possible.
func (dc *dockerCheck) Close() error {
	dc.cancel()
	dc.wg.Wait()

	if closer, ok := dc.client.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("close docker client: %w", err)
		}
	}

	return nil
}
