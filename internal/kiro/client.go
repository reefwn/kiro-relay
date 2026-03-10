package kiro

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\[[\x20-\x3f]*[\x40-\x7e]|\x1b\][^\x07]*\x07|\x1b[()][A-Z0-9]|[\x00-\x09\x0b\x0c\x0e-\x1f]`)

var skipPrefixes = []string{
	"welcome to kiro",
	"💡",
	"model:",
	"all tools are now trusted",
	"agents can sometimes",
	"learn more at",
	"[?",
}

var (
	creditsRe  = regexp.MustCompile(`(?i)credits:\s*[\d.]+`)
	contextRe  = regexp.MustCompile(`(?i)(\d+)%.*context`)
	timeRe     = regexp.MustCompile(`(?i)time:\s*[\d.]+\s*[sm]`)
	statusRe   = regexp.MustCompile(`[▸▹►]`)
)

type Client struct {
	WorkDir    string
	TrustTools string
	Agent      string
	mu         sync.Mutex
}

func NewClient(workDir, trustTools string) *Client {
	return &Client{WorkDir: workDir, TrustTools: trustTools}
}

func (c *Client) SetWorkDir(dir string) {
	c.mu.Lock()
	c.WorkDir = dir
	c.mu.Unlock()
}

func (c *Client) GetWorkDir() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.WorkDir
}

func parseOutput(raw string) string {
	clean := ansiRe.ReplaceAllString(raw, "")
	lines := strings.Split(clean, "\n")

	var response []string
	var credits, ctxPct, elapsed string

	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			if len(response) > 0 {
				response = append(response, "")
			}
			continue
		}

		lower := strings.ToLower(stripped)

		// Extract metadata from status line (▸ Credits: 0.09 • Time: 4s • 25% context)
		if statusRe.MatchString(stripped) || strings.Contains(lower, "credits:") {
			if m := creditsRe.FindString(stripped); m != "" {
				credits = m
			}
			if m := timeRe.FindString(stripped); m != "" {
				elapsed = m
			}
			if m := contextRe.FindStringSubmatch(stripped); len(m) > 1 {
				ctxPct = m[1] + "% context"
			}
			continue
		}

		// Skip kiro-cli chrome
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(lower, prefix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip lines that are just the prompt marker
		if stripped == ">" || strings.HasPrefix(stripped, "> ") {
			stripped = strings.TrimPrefix(stripped, "> ")
			stripped = strings.TrimPrefix(stripped, ">")
			stripped = strings.TrimSpace(stripped)
			if stripped == "" {
				continue
			}
		}

		if stripped != "" {
			response = append(response, stripped)
		}
	}

	text := strings.TrimSpace(strings.Join(response, "\n"))

	// Append metadata footer
	var meta []string
	if credits != "" {
		meta = append(meta, credits)
	}
	if elapsed != "" {
		meta = append(meta, elapsed)
	}
	if ctxPct != "" {
		meta = append(meta, ctxPct)
	}
	if len(meta) > 0 {
		text += "\n\n📊 " + strings.Join(meta, " • ")
	}

	if text == "" {
		return "(empty response)"
	}
	return text
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
	c.mu.Lock()
	agent := c.Agent
	c.mu.Unlock()
	if agent != "" {
		args = append(args, "--agent", agent)
	}
	args = append(args, "--", prompt)

	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	c.mu.Lock()
	cmd.Dir = c.WorkDir
	c.mu.Unlock()

	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "⏱ kiro-cli timed out (5 min limit)", nil
	}

	if err != nil {
		clean := strings.TrimSpace(ansiRe.ReplaceAllString(string(out), ""))
		if clean != "" {
			return clean, nil
		}
		return "", fmt.Errorf("kiro-cli: %w", err)
	}

	return parseOutput(string(out)), nil
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

func (c *Client) ListAgents() (string, error) {
	cmd := exec.Command("kiro-cli", "agent", "list")
	cmd.Dir = c.WorkDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list-agents: %w", err)
	}
	raw := strings.TrimSpace(ansiRe.ReplaceAllString(string(out), ""))
	
	c.mu.Lock()
	currentAgent := c.Agent
	c.mu.Unlock()
	
	// Parse and format: extract name and scope only
	var result []string
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		// Skip header lines and empty lines
		if line == "" || strings.HasPrefix(line, "Workspace:") || strings.HasPrefix(line, "Global:") {
			continue
		}
		
		// Lines with continuation (indented) should be skipped
		if strings.HasPrefix(line, "                    ") {
			continue
		}
		
		// Agent lines start with * or spaces
		if !strings.HasPrefix(line, "*") && !strings.HasPrefix(line, " ") {
			continue
		}
		
		// Extract agent name and scope (first two columns)
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			name := strings.TrimPrefix(fields[0], "*")
			scope := fields[1]
			
			// Add * if this is the current agent
			prefix := "  "
			if currentAgent == name || (currentAgent == "" && strings.HasPrefix(line, "*")) {
				prefix = "* "
			}
			result = append(result, fmt.Sprintf("%s%s %s", prefix, name, scope))
		}
	}
	
	if len(result) == 0 {
		return raw, nil
	}
	return strings.Join(result, "\n"), nil
}

func (c *Client) SetAgent(agent string) {
	c.mu.Lock()
	c.Agent = agent
	c.mu.Unlock()
}

func (c *Client) GetAgent() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Agent
}
