package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRouter(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantCmd string
		wantErr bool
	}{
		{"no args shows help", []string{}, "", true},
		{"mcp command", []string{"mcp"}, "mcp", false},
		{"server command", []string{"server"}, "server", false},
		{"doctor command", []string{"doctor"}, "doctor", false},
		{"unknown command errors", []string{"foo"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			var executedCmd string

			router.Register(&Command{
				Name: "mcp",
				Run: func(ctx context.Context, args []string) error {
					executedCmd = "mcp"
					return nil
				},
			})
			router.Register(&Command{
				Name: "server",
				Run: func(ctx context.Context, args []string) error {
					executedCmd = "server"
					return nil
				},
			})
			router.Register(&Command{
				Name: "doctor",
				Run: func(ctx context.Context, args []string) error {
					executedCmd = "doctor"
					return nil
				},
			})

			err := router.Run(context.Background(), tt.args)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCmd, executedCmd)
			}
		})
	}
}

func TestHelpOutput(t *testing.T) {
	output := captureOutput(func() {
		router := NewRouter()
		_ = router.Run(context.Background(), []string{})
	})

	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "mcp")
	assert.Contains(t, output, "server")
	assert.Contains(t, output, "doctor")
}

func TestHelpFlag(t *testing.T) {
	router := NewRouter()

	tests := []struct {
		name string
		args []string
	}{
		{"help command", []string{"help"}},
		{"-h flag", []string{"-h"}},
		{"--help flag", []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := router.Run(context.Background(), tt.args)
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Usage:")
		})
	}
}

func TestVersionCommand(t *testing.T) {
	router := NewRouter()

	output := captureOutput(func() {
		err := router.Run(context.Background(), []string{"version"})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "trindex version")
}

func TestGlobalFlags(t *testing.T) {
	router := NewRouter()
	var receivedArgs []string

	router.Register(&Command{
		Name: "test",
		Run: func(ctx context.Context, args []string) error {
			receivedArgs = args
			return nil
		},
	})

	args := []string{"test", "--config", "/path/to/config", "--json", "arg1", "arg2"}
	err := router.Run(context.Background(), args)

	assert.NoError(t, err)
	assert.Equal(t, []string{"arg1", "arg2"}, receivedArgs)

	globals := router.GetGlobalFlags()
	assert.Equal(t, "/path/to/config", globals.ConfigPath)
	assert.True(t, globals.JSONOutput)
}

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	return string(buf[:n])
}
