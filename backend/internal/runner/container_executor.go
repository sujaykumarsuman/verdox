package runner

import (
	"archive/tar"
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
	"github.com/sujaykumarsuman/verdox/backend/internal/queue"
)

// ContainerExecutor runs tests in ephemeral Docker containers.
type ContainerExecutor struct {
	docker           client.APIClient
	rdb              *redis.Client
	queue            *queue.RedisQueue
	parser           *Parser
	cfg              *config.Config
	log              zerolog.Logger
	mu               sync.Mutex
	activeContainers map[string]string // runID -> containerID
}

func NewContainerExecutor(cfg *config.Config, rdb *redis.Client, q *queue.RedisQueue, log zerolog.Logger) *ContainerExecutor {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error().Err(err).Msg("failed to create Docker client — container executor disabled")
		return &ContainerExecutor{
			rdb:              rdb,
			queue:            q,
			parser:           NewParser(),
			cfg:              cfg,
			log:              log,
			activeContainers: make(map[string]string),
		}
	}
	return &ContainerExecutor{
		docker:           cli,
		rdb:              rdb,
		queue:            q,
		parser:           NewParser(),
		cfg:              cfg,
		log:              log,
		activeContainers: make(map[string]string),
	}
}

func (e *ContainerExecutor) Supports(mode string) bool {
	return mode == ModeContainer
}

func (e *ContainerExecutor) Execute(ctx context.Context, job *ExecutionJob) (*ExecutionResult, error) {
	if e.docker == nil {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Docker client not available"}, nil
	}

	// Select image and command
	verdoxCfg, _ := LoadVerdoxConfig(job.LocalPath, job.ConfigPath)
	var suiteCfg *SuiteConfig
	if verdoxCfg != nil {
		for i := range verdoxCfg.Suites {
			if verdoxCfg.Suites[i].Type == job.SuiteType {
				suiteCfg = &verdoxCfg.Suites[i]
				break
			}
		}
	}

	dockerImage := e.selectImage(job, suiteCfg)
	testCommand := e.buildCommand(job, suiteCfg)

	e.log.Info().Str("image", dockerImage).Str("command", testCommand).Str("run_id", job.RunID.String()).Msg("creating container")

	// Create container
	containerID, err := e.createContainer(ctx, job, testCommand, dockerImage)
	if err != nil {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Create container: " + err.Error()}, nil
	}

	// Ensure cleanup
	defer func() {
		e.removeContainer(ctx, containerID, job.RunID.String())
	}()

	// Start container
	if err := e.docker.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Start container: " + err.Error()}, nil
	}

	// Cancel listener
	cancelSub := e.queue.SubscribeCancel(ctx, job.RunID.String())
	defer cancelSub.Close()
	go func() {
		ch := cancelSub.Channel()
		select {
		case <-ch:
			e.log.Info().Str("run_id", job.RunID.String()).Msg("cancel signal received")
			e.docker.ContainerKill(context.Background(), containerID, "SIGKILL")
		case <-ctx.Done():
		}
	}()

	// Stream logs
	output, err := e.streamLogs(ctx, containerID, job.RunID.String())
	if err != nil {
		e.log.Warn().Err(err).Msg("failed to stream logs")
	}

	// Wait for completion
	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	maxTimeout := time.Duration(e.cfg.RunnerMaxTimeout) * time.Second
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	exitCode, err := e.waitWithTimeout(ctx, containerID, timeout)
	if err != nil {
		errMsg := "Container execution failed"
		if ctx.Err() != nil {
			errMsg = fmt.Sprintf("Test run exceeded timeout of %d seconds", job.TimeoutSeconds)
		}
		return &ExecutionResult{ExitCode: -1, Output: output, Status: "failed", ErrorMsg: errMsg}, nil
	}

	// Try to read results.json from container before removal
	resultsJSON := e.readResultsFile(ctx, containerID)

	// Parse results
	results := e.parser.ParseWithResultsFile(resultsJSON, output, exitCode)

	// Determine status
	status := "passed"
	for _, r := range results {
		if r.Status == "fail" || r.Status == "error" {
			status = "failed"
			break
		}
	}
	if exitCode != 0 && status == "passed" {
		status = "failed"
	}

	return &ExecutionResult{
		ExitCode: exitCode,
		Output:   output,
		Results:  results,
		Status:   status,
	}, nil
}

func (e *ContainerExecutor) Cancel(ctx context.Context, runID string) error {
	return e.queue.PublishCancel(ctx, runID)
}

func (e *ContainerExecutor) createContainer(ctx context.Context, job *ExecutionJob, testCommand, dockerImage string) (string, error) {
	containerName := fmt.Sprintf("verdox-run-%s", job.RunID.String()[:12])

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
		fmt.Sprintf("VERDOX_RUN_ID=%s", job.RunID.String()),
	}
	for k, v := range job.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Network mode
	networkMode := "none"
	if job.SuiteType == "integration" {
		networkMode = "bridge"
	}

	// Resource limits
	memLimit := int64(2 * 1024 * 1024 * 1024)
	cpuQuota := int64(200000)
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
			"/tmp": "size=5368709120",
		},
	}

	resp, err := e.docker.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	e.mu.Lock()
	e.activeContainers[job.RunID.String()] = resp.ID
	e.mu.Unlock()

	return resp.ID, nil
}

func (e *ContainerExecutor) streamLogs(ctx context.Context, containerID, runID string) (string, error) {
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

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line != "" {
			e.queue.AppendLog(ctx, runID, line)
		}
	}

	return output, nil
}

func (e *ContainerExecutor) waitWithTimeout(ctx context.Context, containerID string, timeout time.Duration) (int64, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statusCh, errCh := e.docker.ContainerWait(waitCtx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
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

// readResultsFile extracts /tmp/verdox-results.json from the container if it exists.
func (e *ContainerExecutor) readResultsFile(ctx context.Context, containerID string) []byte {
	reader, _, err := e.docker.CopyFromContainer(ctx, containerID, ResultsFilePath)
	if err != nil {
		return nil // File doesn't exist — fall back to stdout parsing
	}
	defer reader.Close()

	// Docker returns a tar archive
	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err != nil {
			return nil
		}
		if header.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil
			}
			return data
		}
	}
}

func (e *ContainerExecutor) removeContainer(ctx context.Context, containerID, runID string) {
	e.mu.Lock()
	delete(e.activeContainers, runID)
	e.mu.Unlock()

	removeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	e.docker.ContainerRemove(removeCtx, containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (e *ContainerExecutor) ForceKillAll() {
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

// selectImage picks the Docker image: job config > verdox.yaml > default.
func (e *ContainerExecutor) selectImage(job *ExecutionJob, suiteCfg *SuiteConfig) string {
	if job.DockerImage != "" {
		return job.DockerImage
	}
	if suiteCfg != nil && suiteCfg.Image != "" {
		return suiteCfg.Image
	}
	return e.cfg.RunnerDefaultImage
}

// buildCommand picks the test command: job config > verdox.yaml > default.
func (e *ContainerExecutor) buildCommand(job *ExecutionJob, suiteCfg *SuiteConfig) string {
	if job.TestCommand != "" {
		return job.TestCommand
	}
	if suiteCfg != nil && suiteCfg.Command != "" {
		return suiteCfg.Command
	}
	return "echo 'No test command configured' && exit 1"
}
