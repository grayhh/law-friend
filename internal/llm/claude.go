package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ClaudeCLI struct{}

func (c *ClaudeCLI) Kind() CLIKind { return CLIClaude }

func (c *ClaudeCLI) Ask(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude CLI failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
