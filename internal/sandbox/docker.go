package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog"
)

type DockerSandbox struct {
	cli    *client.Client
	logger *zerolog.Logger
}

func NewDockerSandbox(logger *zerolog.Logger) (*DockerSandbox, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerSandbox{cli: cli, logger: logger}, nil
}

func (s *DockerSandbox) Run(ctx context.Context, cfg RunConfig) (*Result, error) {
	// Security: Limit PID count to prevent fork bombs
	pidsLimit := int64(64)

	// 1. Create container with security hardening
	resp, err := s.cli.ContainerCreate(ctx, &container.Config{
		Image:           cfg.Image,
		Cmd:             []string{"sleep", "infinity"}, // Keep it alive while we compile
		Tty:             false,
		OpenStdin:       true,
		StdinOnce:       true,
		NetworkDisabled: true,
		WorkingDir:      "/home/sandbox",
		User:            "nobody",
	}, &container.HostConfig{
		Resources: container.Resources{
			Memory:     int64(cfg.MemoryLimitKb * 1024),
			MemorySwap: int64(cfg.MemoryLimitKb * 1024), // No swap allowed
			CPUQuota:   100000,                          // 1 CPU
			PidsLimit:  &pidsLimit,                      // Prevent fork bombs
		},
		NetworkMode: "none",
		// Note: ReadonlyRootfs disabled because CopyToContainer doesn't work with it
		// Security is maintained through tmpfs mounts, no network, and dropped capabilities
		SecurityOpt: []string{"no-new-privileges"},
		CapDrop:     []string{"ALL"},
		Tmpfs: map[string]string{
			"/home/sandbox": "rw,exec,nosuid,size=64m,mode=1777",
			"/tmp":          "rw,noexec,nosuid,size=16m,mode=1777",
		},
	}, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	defer s.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})

	// 2. Start container
	if err := s.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// 3. Write source code using exec (CopyToContainer doesn't work with tmpfs mounts)
	writeCmd := []string{"sh", "-c", fmt.Sprintf("cat > /home/sandbox/%s", cfg.SourceFile)}
	execResp, err := s.cli.ContainerExecCreate(ctx, resp.ID, container.ExecOptions{
		Cmd:         writeCmd,
		AttachStdin: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create write exec: %w", err)
	}

	attachResp, err := s.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach write exec: %w", err)
	}

	_, err = attachResp.Conn.Write([]byte(cfg.SourceCode))
	if err != nil {
		attachResp.Close()
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}
	attachResp.CloseWrite()
	attachResp.Close()

	// Wait for write to complete
	for {
		inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect write exec: %w", err)
		}
		if !inspect.Running {
			break
		}
	}

	s.logger.Debug().Str("container", resp.ID).Msg("source code written via exec")

	// 5. Compile if needed
	if len(cfg.CompileCmd) > 0 {
		execResp, err := s.cli.ContainerExecCreate(ctx, resp.ID, container.ExecOptions{
			Cmd:          cfg.CompileCmd,
			WorkingDir:   "/home/sandbox",
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create compile exec: %w", err)
		}

		startResp, err := s.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to start compile exec: %w", err)
		}
		defer startResp.Close()

		var stdout, stderr bytes.Buffer
		_, err = stdcopy.StdCopy(&stdout, &stderr, startResp.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to capture compile logs: %w", err)
		}

		inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect compile exec: %w", err)
		}

		if inspect.ExitCode != 0 {
			return &Result{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: inspect.ExitCode,
			}, nil // Return with compilation error
		}
	}

	// 6. Execute
	startTime := time.Now()
	execResp, err = s.cli.ContainerExecCreate(ctx, resp.ID, container.ExecOptions{
		Cmd:          cfg.RunCmd,
		WorkingDir:   "/home/sandbox",
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create run exec: %w", err)
	}

	startResp, err := s.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start run exec: %w", err)
	}
	defer startResp.Close()

	if cfg.Stdin != "" {
		_, _ = startResp.Conn.Write([]byte(cfg.Stdin))
		_ = startResp.CloseWrite()
	}

	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, err := stdcopy.StdCopy(&stdout, &stderr, startResp.Reader)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("failed to read execution logs: %w", err)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	duration := time.Since(startTime)
	inspect, err := s.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect run exec: %w", err)
	}

	return &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: inspect.ExitCode,
		TimeMs:   duration.Milliseconds(),
		MemoryKb: 0, // Finding actual memory usage is complex in Docker Exec, leaving at 0 for MVP
	}, nil
}

func (s *DockerSandbox) EnsureImage(ctx context.Context, img string) error {
	_, _, err := s.cli.ImageInspectWithRaw(ctx, img)
	if err == nil {
		return nil // Image already exists
	}

	s.logger.Info().Str("image", img).Msg("pulling docker image")
	reader, err := s.cli.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", img, err)
	}
	defer reader.Close()

	// Important: must consume the reader to finish the pull
	_, _ = io.Copy(io.Discard, reader)

	s.logger.Info().Str("image", img).Msg("successfully pulled docker image")
	return nil
}
