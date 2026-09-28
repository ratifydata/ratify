package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/ratifydata/ratify/internal/config"
)

func TestRootCommand(t *testing.T) {
	for _, tt := range []struct {
		name          string
		args          []string
		want, wantErr string
	}{
		{name: "welcome", want: "Welcome to ratify, A data contract workflow engine\n"},
		{name: "help", args: []string{"--help"}, want: "A command line tool for data contract workflow engine"},
		{name: "connect help", args: []string{"connect", "--help"}, want: "ratify connect [command]"},
		{name: "auth help", args: []string{"auth", "--help"}, want: "set-key"},
		{name: "unknown command", args: []string{"unknown"}, wantErr: `unknown command "unknown"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			previous := orgID
			t.Cleanup(func() { orgID = previous })
			cmd := newRootCmd(&config.Config{}, nil)
			got, err := executeCLI(t, cmd, tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Execute() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("output = %q, want containing %q", got, tt.want)
			}
		})
	}
}

func TestRootCommandRegistersCommands(t *testing.T) {
	previous := orgID
	t.Cleanup(func() { orgID = previous })
	root := newRootCmd(&config.Config{}, nil)
	for _, path := range [][]string{{"auth", "set-key"}, {"connect", "list"}, {"connect", "test"}, {"connect", "add"}} {
		t.Run(strings.Join(path, "/"), func(t *testing.T) {
			cmd, remaining, err := root.Find(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(remaining) != 0 || cmd.Name() != path[len(path)-1] {
				t.Errorf("Find(%v) = %q, remaining %v", path, cmd.Name(), remaining)
			}
		})
	}
}

// Execute exits the process on errors, so exercise its public entry point in a
// subprocess instead of replacing os.Exit or terminating the test runner.
func TestExecute(t *testing.T) {
	if mode := os.Getenv("RATIFY_CLI_EXECUTE_TEST"); mode != "" {
		os.Args = []string{"ratify"}
		if mode == "failure" {
			os.Args = append(os.Args, "unknown")
		}
		Execute(&config.Config{}, nil)
		return
	}
	for _, tt := range []struct {
		name string
		code int
		want string
	}{
		{"success", 0, "Welcome to ratify"},
		{"failure", 1, `unknown command "unknown"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestExecute$")
			cmd.Env = append(os.Environ(), "RATIFY_CLI_EXECUTE_TEST="+tt.name)
			output, err := cmd.CombinedOutput()
			if cmd.ProcessState == nil {
				t.Fatalf("start subprocess: %v", err)
			}
			if got := cmd.ProcessState.ExitCode(); got != tt.code {
				t.Errorf("exit code = %d, want %d; output: %s", got, tt.code, output)
			}
			if !strings.Contains(string(output), tt.want) {
				t.Errorf("output = %q, want containing %q", output, tt.want)
			}
		})
	}
}
