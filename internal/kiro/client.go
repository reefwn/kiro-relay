package kiro

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|[\x00-\x09\x0b\x0c\x0e-\x1f]`)

type Client struct {
	WorkDir    string
	TrustTools string
}

func NewClient(workDir, trustTools string) *Client {
	return &Client{WorkDir: workDir, TrustTools: trustTools}
}

func (c *Client) Run(prompt string, resume bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{"chat", "--no-interactive"}
	if c.TrustTools == "*" {
		args = append(args, "--trust-all-tools")
	} else if c.TrustTools != "" {
		args = append(args, "--trust-tools", c.TrustTools)
	}
	if resume {
		args = append(args, "--resume")
	}
	args = append(args, "--", prompt)

	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	cmd.Dir = c.WorkDir

	out, err := cmd.CombinedOutput()
	clean := strings.TrimSpace(ansiRe.ReplaceAllString(string(out), ""))

	if ctx.Err() == context.DeadlineExceeded {
		return "⏱ kiro-cli timed out (5 min limit)", nil
	}

	if err != nil {
		if clean != "" {
			return clean, nil
		}
		return "", fmt.Errorf("kiro-cli: %w", err)
	}
	if clean == "" {
		return "(empty response)", nil
	}
	return clean, nil
}

func (c *Client) ListSessions() (string, error) {
	cmd := exec.Command("kiro-cli", "chat", "--list-sessions")
	cmd.Dir = c.WorkDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list-sessions: %w", err)
	}
	return strings.TrimSpace(ansiRe.ReplaceAllString(string(out), "")), nil
}

func (c *Client) DeleteSession(id string) error {
	cmd := exec.Command("kiro-cli", "chat", "--delete-session", id)
	cmd.Dir = c.WorkDir
	return cmd.Run()
}
