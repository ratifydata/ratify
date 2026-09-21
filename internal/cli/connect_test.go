package cli

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestPrompt(t *testing.T) {
	for _, tt := range []struct {
		name, input, want string
		wantErr           error
	}{
		{"text", "localhost\n", "localhost", nil},
		{"whitespace", " \tlocalhost \t\n", "localhost", nil},
		{"windows newline", "localhost\r\n", "localhost", nil},
		{"blank", "\n", "", nil},
		{"empty input", "", "", io.EOF},
		{"unterminated input", "localhost", "", io.EOF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prompt(bufio.NewReader(strings.NewReader(tt.input)), "Hostname")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("prompt() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("prompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromptReadsOneLine(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("localhost\n5432\n"))
	for _, want := range []string{"localhost", "5432"} {
		got, err := prompt(reader, "Value")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("prompt() = %q, want %q", got, want)
		}
	}
}

func TestConnectCommandValidation(t *testing.T) {
	// Validation must fail before any database access; a nil inspector makes
	// accidental execution of a database operation fail the test.
	for _, tt := range []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"missing organization", []string{"list"}, `required flag(s) "org-id" not set`},
		{"missing connection", []string{"--org-id", "org", "test"}, "accepts 1 arg(s), received 0"},
		{"extra connection", []string{"--org-id", "org", "test", "one", "two"}, "accepts 1 arg(s), received 2"},
		{"invalid connection", []string{"--org-id", "org", "test", "not-a-uuid"}, "invalid connection ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			previousOrgID := orgID
			t.Cleanup(func() { orgID = previousOrgID })
			connect := &ConnectCmd{}
			cmd := connect.Connect()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Execute() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
