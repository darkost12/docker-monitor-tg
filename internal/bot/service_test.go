package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"docker-tg-bot/internal/docker"
)

type fakeDocker struct {
	containers []docker.Container
	logs       string
	err        error

	logsCalls []logsCall
}

type logsCall struct {
	name string
	tail int
}

func (f *fakeDocker) Containers(ctx context.Context) ([]docker.Container, error) {
	return f.containers, f.err
}

func (f *fakeDocker) Logs(ctx context.Context, name string, tail int) (string, error) {
	f.logsCalls = append(f.logsCalls, logsCall{name, tail})
	return f.logs, f.err
}

func newSvc(f *fakeDocker) *Service { return NewService(f, 42) }

func TestAuthorized(t *testing.T) {
	s := newSvc(&fakeDocker{})
	if !s.Authorized(42) {
		t.Error("allowed user should pass")
	}
	for _, id := range []int64{0, 41, 43, -42} {
		if s.Authorized(id) {
			t.Errorf("user %d is not allowed", id)
		}
	}
}

func TestHandleCommandHelp(t *testing.T) {
	s := newSvc(&fakeDocker{})
	for _, cmd := range []string{"start", "help"} {
		got := s.HandleCommand(context.Background(), cmd, nil)
		if !strings.Contains(got, "/ps") || !strings.Contains(got, "/logs") {
			t.Errorf("/%s: no data, got %q", cmd, got)
		}
	}
}

func TestHandleCommandUnknown(t *testing.T) {
	s := newSvc(&fakeDocker{})
	got := s.HandleCommand(context.Background(), "rm", nil)
	if !strings.Contains(got, "Unknown command") || !strings.Contains(got, "/ps") {
		t.Errorf("got %q", got)
	}
}

func TestListContainers(t *testing.T) {
	f := &fakeDocker{containers: []docker.Container{
		{ID: "abc123", Name: "web", Image: "nginx:latest", Status: "Up 2 hours"},
		{ID: "def456", Name: "db<1>", Image: "postgres", Status: "Up"},
	}}
	s := newSvc(f)

	for _, cmd := range []string{"ps", "containers"} {
		got := s.HandleCommand(context.Background(), cmd, nil)
		for _, want := range []string{
			"<code>abc123</code>", "<b>web</b>", "nginx:latest — Up 2 hours",
			"<b>db&lt;1&gt;</b>", // HTML is escaped
		} {
			if !strings.Contains(got, want) {
				t.Errorf("/%s: нет %q в %q", cmd, want, got)
			}
		}
	}
}

func TestListContainersEmptyAndError(t *testing.T) {
	got := newSvc(&fakeDocker{}).HandleCommand(context.Background(), "ps", nil)
	if got != "No running containers." {
		t.Errorf("empty list: %q", got)
	}

	got = newSvc(&fakeDocker{err: errors.New("daemon <down>")}).HandleCommand(context.Background(), "ps", nil)
	if got != "Error: daemon &lt;down&gt;" {
		t.Errorf("error docker: %q", got)
	}
}

func TestLogsDefaultTail(t *testing.T) {
	f := &fakeDocker{logs: "hello\nworld\n"}
	got := newSvc(f).HandleCommand(context.Background(), "logs", []string{"web"})

	if got != "<pre>hello\nworld\n</pre>" {
		t.Errorf("got %q", got)
	}
	if len(f.logsCalls) != 1 || f.logsCalls[0] != (logsCall{"web", defaultTail}) {
		t.Errorf("docker calls: %+v", f.logsCalls)
	}
}

func TestLogsTailArgument(t *testing.T) {
	tests := []struct {
		arg  string
		want int
	}{
		{"1", 1},
		{"200", 200},
		{"1000", 1000},
		{"5000", maxTail},
	}
	for _, tt := range tests {
		f := &fakeDocker{logs: "x"}
		newSvc(f).HandleCommand(context.Background(), "logs", []string{"web", tt.arg})
		if len(f.logsCalls) != 1 || f.logsCalls[0].tail != tt.want {
			t.Errorf("N=%s: calls %+v, want tail=%d", tt.arg, f.logsCalls, tt.want)
		}
	}
}

func TestLogsBadInputDoesNotCallDocker(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments", nil, "Specify container name"},
		{"flag", []string{"--help"}, "Invalid container name"},
		{"injection", []string{"web;rm"}, "Invalid container name"},
		{"N not a number", []string{"web", "abc"}, "positive number"},
		{"N is zero", []string{"web", "0"}, "positive number"},
		{"N is negative", []string{"web", "-5"}, "positive number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeDocker{logs: "secret"}
			got := newSvc(f).HandleCommand(context.Background(), "logs", tt.args)
			if !strings.Contains(got, tt.want) {
				t.Errorf("got %q, expected %q", got, tt.want)
			}
			if len(f.logsCalls) != 0 {
				t.Errorf("docker should not be called: %+v", f.logsCalls)
			}
		})
	}
}

func TestLogsEscapesHTML(t *testing.T) {
	f := &fakeDocker{logs: "<script>alert(1)</script> & done"}
	got := newSvc(f).HandleCommand(context.Background(), "logs", []string{"web"})

	want := "<pre>&lt;script&gt;alert(1)&lt;/script&gt; &amp; done</pre>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLogsEmpty(t *testing.T) {
	got := newSvc(&fakeDocker{logs: " \n\n"}).HandleCommand(context.Background(), "logs", []string{"web"})
	if got != "Logs empty." {
		t.Errorf("got %q", got)
	}
}

func TestLogsTruncatesKeepingNewest(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 2000; i++ {
		sb.WriteString("log line number ")
		sb.WriteString(strings.Repeat("x", 5))
		sb.WriteString("\n")
	}
	sb.WriteString("LAST")
	f := &fakeDocker{logs: sb.String()}

	got := newSvc(f).HandleCommand(context.Background(), "logs", []string{"web"})

	if !strings.Contains(got, "LAST") {
		t.Error("most recent line should be present")
	}
	if !strings.HasPrefix(got, "<pre>…\n") {
		t.Errorf("truncation marker expected, start: %q", got[:20])
	}
	if n := len([]rune(got)); n > 4096 {
		t.Errorf("response %d symbols, Telegram limit 4096", n)
	}
}

func TestLogsDockerErrors(t *testing.T) {
	got := newSvc(&fakeDocker{err: docker.ErrTimeout}).HandleCommand(context.Background(), "logs", []string{"web"})
	if !strings.Contains(got, "timeout") {
		t.Errorf("got %q", got)
	}

	got = newSvc(&fakeDocker{err: docker.ErrInvalidName}).HandleCommand(context.Background(), "logs", []string{"web"})
	if got != "Invalid container name or hash." {
		t.Errorf("got %q", got)
	}
}

func TestHandleTextActsAsLogs(t *testing.T) {
	f := &fakeDocker{logs: "ok"}
	got := newSvc(f).HandleText(context.Background(), "  web   30 ")

	if got != "<pre>ok</pre>" {
		t.Errorf("got %q", got)
	}
	if len(f.logsCalls) != 1 || f.logsCalls[0] != (logsCall{"web", 30}) {
		t.Errorf("docker calls: %+v", f.logsCalls)
	}
}

func TestHandleTextEmpty(t *testing.T) {
	f := &fakeDocker{}
	got := newSvc(f).HandleText(context.Background(), "   ")
	if !strings.Contains(got, "Specify container name") || len(f.logsCalls) != 0 {
		t.Errorf("got %q, calls %+v", got, f.logsCalls)
	}
}
