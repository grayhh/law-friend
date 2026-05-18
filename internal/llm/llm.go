package llm

import (
	"context"
	"fmt"
	"os/exec"
)

type CLIKind string

const (
	CLIClaude CLIKind = "claude"
	CLICodex  CLIKind = "codex"
)

type Client interface {
	Kind() CLIKind
	Ask(ctx context.Context, prompt string) (string, error)
}

func Detect() []CLIKind {
	var available []CLIKind
	if _, err := exec.LookPath("claude"); err == nil {
		available = append(available, CLIClaude)
	}
	if _, err := exec.LookPath("codex"); err == nil {
		available = append(available, CLICodex)
	}
	return available
}

func New(kind CLIKind) (Client, error) {
	switch kind {
	case CLIClaude:
		return &ClaudeCLI{}, nil
	case CLICodex:
		return &CodexCLI{}, nil
	default:
		return nil, fmt.Errorf("unknown CLI kind: %s", kind)
	}
}
