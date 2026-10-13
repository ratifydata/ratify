package cli

import (
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	for _, tt := range []struct {
		name          string
		args          []string
		want, wantErr string
	}{
		{name: "help", args: []string{"--help"}, want: "A command line tool for data contract workflow engine"},
		{name: "connect help", args: []string{"connect", "--help"}, want: "ratify connect [command]"},
		{name: "auth help", args: []string{"auth", "--help"}, want: "set-key"},
		{name: "unknown command", args: []string{"unknown"}, wantErr: `unknown command "unknown"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
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
	root := newRootCmd()
	for _, path := range [][]string{{"auth", "set-key"}, {"connect", "list"}, {"connect", "test"}, {"connect", "add"}, {"team", "add"}, {"team", "list"}} {
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
