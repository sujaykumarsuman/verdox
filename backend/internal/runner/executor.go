package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/queue"
)

type Executor struct {
	docker           client.APIClient
	rdb              *redis.Client
	queue            *queue.RedisQueue
	cfg              *config.Config
	log              zerolog.Logger
	mu               sync.Mutex
	activeContainers map[string]string // runID -> containerID
}

func NewExecutor(cfg *config.Config, rdb *redis.Client, q *queue.RedisQueue, log zerolog.Logger) *Executor {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error().Err(err).Msg("failed to create Docker client — runner will not be functional")
		return &Executor{
			rdb:              rdb,
			queue:            q,
			cfg:              cfg,
			log:              log,
			activeContainers: make(map[string]string),
		}
	}
	return &Executor{
		docker:           cli,
		rdb:              rdb,
		queue:            q,
		cfg:              cfg,
		log:              log,
		activeContainers: make(map[string]string),
	}
}

func (e *Executor) CreateContainer(ctx context.Context, job *model.JobPayload, testCommand, dockerImage string) (string, error) {
	if e.docker == nil {
		return "", fmt.Errorf("docker client not available")
	}

	containerName := fmt.Sprintf("verdox-run-%s", job.TestRunID[:12])

	// Pull image if needed
	_, _, err := e.docker.ImageInspectWithRaw(ctx, dockerImage)
	if err != nil {
		e.log.Info().Str("image", dockerImage).Msg("pulling Docker image")
		reader, err := e.docker.ImagePull(ctx, dockerImage, image.PullOptions{})
		if err != nil {
			return "", fmt.Errorf("pull image %s: %w", dockerImage, err)
		}
		io.Copy(io.Discard, reader)
		reader.Close()
	}

	// Environment variables
	env := []string{
		"CI=true",
		fmt.Sprintf("VERDOX_RUN_ID=%s", job.TestRunID),
	}

	// Network mode — none for unit tests, default for integration
	networkMode := "none"
	if job.TestType == "integration" {
		networkMode = "bridge"
	}

	// Resource limits
	memLimit := int64(2 * 1024 * 1024 * 1024) // 2GB
	cpuQuota := int64(200000)                   // 2 CPUs (100000 per CPU)
	cpuPeriod := int64(100000)
	pidsLimit := int64(256)

	containerConfig := &container.Config{
		Image:      dockerImage,
		Cmd:        []string{"sh", "-c", testCommand},
		Env:        env,
		WorkingDir: "/workspace",
	}

	hostConfig := &container.HostConfig{
		Binds:       []string{fmt.Sprintf("%s:/workspace:ro", job.LocalPath)},
		NetworkMode: container.NetworkMode(networkMode),
		Resources: container.Resources{
			Memory:    memLimit,
			CPUQuota:  cpuQuota,
			CPUPeriod: cpuPeriod,
			PidsLimit: &pidsLimit,
		},
		Tmpfs: map[string]string{
			"/tmp": "size=5368709120", // 5GB tmpfs
		},
	}

	resp, err := e.docker.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	e.mu.Lock()
	e.activeContainers[job.TestRunID] = resp.ID
	e.mu.Unlock()

	return resp.ID, nil
}

func (e *Executor) StartContainer(ctx context.Context, containerID string) error {
	if e.docker == nil {
		return fmt.Errorf("docker client not available")
	}
	return e.docker.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (e *Executor) StreamLogs(ctx context.Context, containerID, runID string) (string, error) {
	if e.docker == nil {
		return "", fmt.Errorf("docker client not available")
	}

	reader, err := e.docker.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return "", fmt.Errorf("attach logs: %w", err)
	}
	defer reader.Close()

	var stdout, stderr bytes.Buffer
	stdcopy.StdCopy(&stdout, &stderr, reader)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	// Append to Redis log buffer
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line != "" {
			e.queue.AppendLog(ctx, runID, line)
		}
	}

	return output, nil
}

func (e *Executor) WaitWithTimeout(ctx context.Context, containerID string, timeout time.Duration) (int64, error) {
	if e.docker == nil {
		return -1, fmt.Errorf("docker client not available")
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statusCh, errCh := e.docker.ContainerWait(waitCtx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			// Timeout or context cancelled — kill the container
			killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer killCancel()
			e.docker.ContainerKill(killCtx, containerID, "SIGKILL")
			return -1, fmt.Errorf("wait failed: %w", err)
		}
	case status := <-statusCh:
		return status.StatusCode, nil
	}

	return -1, fmt.Errorf("unexpected wait state")
}

func (e *Executor) RemoveContainer(ctx context.Context, containerID string, runID string) error {
	if e.docker == nil {
		return nil
	}

	e.mu.Lock()
	delete(e.activeContainers, runID)
	e.mu.Unlock()

	removeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return e.docker.ContainerRemove(removeCtx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (e *Executor) ForceKillAll() {
	e.mu.Lock()
	containers := make(map[string]string)
	for k, v := range e.activeContainers {
		containers[k] = v
	}
	e.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for runID, containerID := range containers {
		e.log.Warn().Str("run_id", runID).Str("container_id", containerID).Msg("force killing container")
		if e.docker != nil {
			e.docker.ContainerKill(ctx, containerID, "SIGKILL")
			e.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true, RemoveVolumes: true})
		}
	}
}

// SelectImage picks the Docker image based on config, test type, or defaults.
func SelectImage(job *model.JobPayload, suiteCfg *SuiteConfig, defaultImage string) string {
	if suiteCfg != nil && suiteCfg.Image != "" {
		return suiteCfg.Image
	}

	// Auto-detect by test type
	switch job.TestType {
	case "unit":
		return "golang:1.26-alpine"
	case "integration":
		return "golang:1.26-alpine"
	}

	return defaultImage
}

// BuildTestCommand constructs the shell command to run tests.
func BuildTestCommand(job *model.JobPayload, suiteCfg *SuiteConfig) string {
	if suiteCfg != nil && suiteCfg.Command != "" {
		return suiteCfg.Command
	}

	// Default commands by test type
	switch job.TestType {
	case "unit":
		return "go test -v -json ./..."
	case "integration":
		return "go test -v -json -tags=integration ./..."
	}

	return "echo 'No test command configured' && exit 1"
}
