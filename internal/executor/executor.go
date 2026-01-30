package executor

import (
	"context"
	"fmt"

	"github.com/itstheanurag/executioner/internal/languages"
	"github.com/itstheanurag/executioner/internal/sandbox"
)

type ExecutionResult struct {
	Status    string
	Stdout    string
	Stderr    string
	ExitCode  int
	TimeMs    int64
	MemoryKb  int64
	ErrorType string
}

type Executor struct {
	registry *languages.Registry
	sandbox  sandbox.Sandbox
}

func NewExecutor(registry *languages.Registry, sb sandbox.Sandbox) *Executor {
	return &Executor{
		registry: registry,
		sandbox:  sb,
	}
}

type ExecuteOptions struct {
	LanguageID    string
	SourceCode    string
	Stdin         string
	TimeLimitMs   int
	MemoryLimitKb int
}

func (e *Executor) Execute(ctx context.Context, opts ExecuteOptions) (*ExecutionResult, error) {
	lang, err := e.registry.Get(opts.LanguageID)
	if err != nil {
		return &ExecutionResult{
			Status:    "error",
			ErrorType: "Invalid Language",
		}, nil
	}

	res, err := e.sandbox.Run(ctx, sandbox.RunConfig{
		Image:         lang.Config.Image,
		SourceCode:    opts.SourceCode,
		SourceFile:    lang.Config.SourceFile,
		CompileCmd:    lang.Config.CompileCommand,
		RunCmd:        lang.Config.RunCommand,
		Stdin:         opts.Stdin,
		TimeLimitMs:   opts.TimeLimitMs,
		MemoryLimitKb: opts.MemoryLimitKb,
	})

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &ExecutionResult{
				Status:    "error",
				ErrorType: "Time Limit Exceeded",
			}, nil
		}
		return nil, fmt.Errorf("sandbox execution failed: %w", err)
	}

	status := "success"
	if res.ExitCode != 0 {
		status = "runtime_error"
	}

	return &ExecutionResult{
		Status:   status,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: res.ExitCode,
		TimeMs:   res.TimeMs,
		MemoryKb: res.MemoryKb,
	}, nil
}
