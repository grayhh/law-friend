package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CodexCLI struct{}

func (c *CodexCLI) Kind() CLIKind { return CLICodex }

func (c *CodexCLI) Ask(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "codex", "exec", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("codex CLI failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
