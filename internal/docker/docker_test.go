package docker

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidName(t *testing.T) {
	valid := []string{"nginx", "my-app", "my_app.1", "a1b2c3d4e5f6", "A", "0abc"}
	invalid := []string{
		"", "-rf", "--help", "-a", "_x", ".x",
		"a b", "a;b", "a|b", "a&&b", "$(id)", "`id`", "a\nb", "a/b", "имя",
	}
	for _, s := range valid {
		if !ValidName(s) {
			t.Errorf("ValidName(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if ValidName(s) {
			t.Errorf("ValidName(%q) = true, want false", s)
		}
	}
}

func TestParseContainers(t *testing.T) {
	out := "abc123\tweb\tnginx:latest\tUp 2 hours\n" +
		"def456\tdb\tpostgres:16\tUp 5 minutes (healthy)\n"
	want := []Container{
		{ID: "abc123", Name: "web", Image: "nginx:latest", Status: "Up 2 hours"},
		{ID: "def456", Name: "db", Image: "postgres:16", Status: "Up 5 minutes (healthy)"},
	}
	if got := parseContainers(out); !reflect.DeepEqual(got, want) {
		t.Errorf("parseContainers = %+v, want %+v", got, want)
	}
}

func TestParseContainersEdgeCases(t *testing.T) {
	if got := parseContainers(""); len(got) != 0 {
		t.Errorf("empty output: got %+v", got)
	}
	if got := parseContainers("\n\n"); len(got) != 0 {
		t.Errorf("empty strings: got %+v", got)
	}
	got := parseContainers("broken\tline\nabc\tweb\timg\tUp\n")
	if len(got) != 1 || got[0].Name != "web" {
		t.Errorf("got %+v", got)
	}
}

func fakeExec(out string, err error, calls *[][]string) *Client {
	return &Client{
		timeout: time.Second,
		exec: func(ctx context.Context, args ...string) ([]byte, error) {
			if calls != nil {
				*calls = append(*calls, args)
			}
			return []byte(out), err
		},
	}
}

func TestContainers(t *testing.T) {
	var calls [][]string
	c := fakeExec("abc\tweb\tnginx\tUp\n", nil, &calls)

	got, err := c.Containers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "web" {
		t.Errorf("got %+v", got)
	}
	if len(calls) != 1 || calls[0][0] != "ps" {
		t.Errorf("docker ps call expected, got %v", calls)
	}
}

func TestLogsPassesArgs(t *testing.T) {
	var calls [][]string
	c := fakeExec("line1\nline2\n", nil, &calls)

	out, err := c.Logs(context.Background(), "web", 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "line1\nline2\n" {
		t.Errorf("out = %q", out)
	}
	want := []string{"logs", "--tail", "25", "web"}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], want) {
		t.Errorf("docker arguments = %v, want %v", calls, want)
	}
}

func TestLogsRejectsInvalidNameWithoutCallingDocker(t *testing.T) {
	var calls [][]string
	c := fakeExec("secret", nil, &calls)

	for _, name := range []string{"--help", "a;b", ""} {
		_, err := c.Logs(context.Background(), name, 10)
		if !errors.Is(err, ErrInvalidName) {
			t.Errorf("Logs(%q): err = %v, want ErrInvalidName", name, err)
		}
	}
	if len(calls) != 0 {
		t.Errorf("docker should not be called, calls: %v", calls)
	}
}

func TestRunWrapsErrorWithOutput(t *testing.T) {
	boom := errors.New("exit status 1")
	c := fakeExec("Error: No such container: nope\n", boom, nil)

	_, err := c.Logs(context.Background(), "nope", 10)
	if err == nil {
		t.Fatal("error expected")
	}
	if !errors.Is(err, boom) {
		t.Errorf("original error should be preserved %%w: %v", err)
	}
	if !strings.Contains(err.Error(), "No such container: nope") {
		t.Errorf("docker output should be in error: %v", err)
	}
}

func TestRunTimeout(t *testing.T) {
	c := &Client{
		timeout: 20 * time.Millisecond,
		exec: func(ctx context.Context, args ...string) ([]byte, error) {
			<-ctx.Done() // stuck docker
			return nil, ctx.Err()
		},
	}

	start := time.Now()
	_, err := c.Containers(context.Background())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("timeout is too late: %v", time.Since(start))
	}
}
