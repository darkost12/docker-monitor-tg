package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	"docker-tg-bot/internal/docker"
	"docker-tg-bot/internal/textutil"
)

const (
	defaultTail   = 50
	maxTail       = 1000
	maxLogsLen    = 3800
	maxListingLen = 4000
)

type Docker interface {
	Containers(ctx context.Context) ([]docker.Container, error)
	Logs(ctx context.Context, name string, tail int) (string, error)
}

type Service struct {
	docker    Docker
	allowedID int64
}

func NewService(d Docker, allowedID int64) *Service {
	return &Service{docker: d, allowedID: allowedID}
}

func (s *Service) Authorized(userID int64) bool { return userID == s.allowedID }

func (s *Service) HandleCommand(ctx context.Context, cmd string, args []string) string {
	switch cmd {
	case "start", "help":
		return helpText
	case "ps", "containers":
		return s.list(ctx)
	case "logs":
		return s.logs(ctx, args)
	default:
		return "Unknown command.\n\n" + helpText
	}
}

func (s *Service) HandleText(ctx context.Context, text string) string {
	return s.logs(ctx, strings.Fields(text))
}

const helpText = "/ps — list of running containers\n" +
	"/logs &lt;name|hash&gt; [N] — last N lines (default 50, max 1000)\n" +
	"Container name or hash can be sent."

func (s *Service) list(ctx context.Context) string {
	containers, err := s.docker.Containers(ctx)
	if err != nil {
		return errorText(err)
	}
	if len(containers) == 0 {
		return "No running containers."
	}

	var sb strings.Builder
	for _, c := range containers {
		fmt.Fprintf(&sb, "<code>%s</code> <b>%s</b>\n%s — %s\n\n",
			html.EscapeString(c.ID), html.EscapeString(c.Name),
			html.EscapeString(c.Image), html.EscapeString(c.Status))
	}
	return textutil.Head(sb.String(), maxListingLen)
}

func (s *Service) logs(ctx context.Context, args []string) string {
	if len(args) == 0 {
		return "Specify container name or hash: /logs &lt;name|hash&gt; [N]"
	}
	name := args[0]
	if !docker.ValidName(name) {
		return "Invalid container name or hash."
	}

	tail := defaultTail
	if len(args) > 1 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			return "N should be a positive number."
		}
		tail = min(n, maxTail)
	}

	out, err := s.docker.Logs(ctx, name, tail)
	if err != nil {
		return errorText(err)
	}
	if strings.TrimSpace(out) == "" {
		return "Logs empty."
	}
	return "<pre>" + html.EscapeString(textutil.Tail(out, maxLogsLen)) + "</pre>"
}

func errorText(err error) string {
	if errors.Is(err, docker.ErrInvalidName) {
		return "Invalid container name or hash."
	}
	return "Error: " + html.EscapeString(err.Error())
}
