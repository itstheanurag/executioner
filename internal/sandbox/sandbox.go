package sandbox

import (
	"context"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimeMs   int64
	MemoryKb int64
}

type Sandbox interface {
	Run(ctx context.Context, config RunConfig) (*Result, error)
	EnsureImage(ctx context.Context, image string) error
}

type RunConfig struct {
	Image         string
	SourceCode    string
	SourceFile    string
	CompileCmd    []string
	RunCmd        []string
	Stdin         string
	TimeLimitMs   int
	MemoryLimitKb int
}
