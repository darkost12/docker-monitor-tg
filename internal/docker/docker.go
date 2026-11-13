package docker

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidName = errors.New("invalid container name or hash")
	ErrTimeout     = errors.New("docker timeout")
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

func ValidName(s string) bool { return nameRe.MatchString(s) }

type CommandError struct {
	Output string
	Err    error
}

func (e *CommandError) Error() string {
	if e.Output != "" {
		return e.Output
	}
	return e.Err.Error()
}

func (e *CommandError) Unwrap() error { return e.Err }

type Container struct {
	ID     string
	Name   string
	Image  string
	Status string
}

type ExecFunc func(ctx context.Context, args ...string) ([]byte, error)

func cliExec(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

type Client struct {
	timeout time.Duration
	exec    ExecFunc
}

func NewClient(timeout time.Duration) *Client {
	return &Client{timeout: timeout, exec: cliExec}
}

func (c *Client) Containers(ctx context.Context) ([]Container, error) {
	out, err := c.run(ctx, "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	if err != nil {
		return nil, err
	}
	return parseContainers(out), nil
}

func (c *Client) Logs(ctx context.Context, name string, tail int) (string, error) {
	if !ValidName(name) {
		return "", ErrInvalidName
	}
	return c.run(ctx, "logs", "--tail", strconv.Itoa(tail), name)
}

func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	out, err := c.exec(ctx, args...)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", ErrTimeout
	}
	if err != nil {
		return "", &CommandError{Output: strings.TrimSpace(string(out)), Err: err}
	}
	return string(out), nil
}

func parseContainers(out string) []Container {
	var list []Container
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		p := strings.SplitN(line, "\t", 4)
		if len(p) < 4 {
			continue
		}
		list = append(list, Container{ID: p[0], Name: p[1], Image: p[2], Status: p[3]})
	}
	return list
}
