package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ExportFlags struct {
	Output    string
	Namespace string
	Since     string
	Until     string
	APIURL    string
	APIKey    string
}

func NewExportCommand() *Command {
	return &Command{
		Name:        "export",
		Description: "Export memories",
		Run: func(ctx context.Context, args []string) error {
			return runExport(ctx, args)
		},
	}
}

func runExport(ctx context.Context, args []string) error {
	flags := &ExportFlags{}
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.StringVar(&flags.Output, "output", "", "Output file (default: stdout)")
	fs.StringVar(&flags.Namespace, "namespace", "", "Export specific namespace")
	fs.StringVar(&flags.Since, "since", "", "Export memories since date (RFC3339)")
	fs.StringVar(&flags.Until, "until", "", "Export memories until date (RFC3339)")
	fs.StringVar(&flags.APIURL, "api-url", getEnv("TRINDEX_API_URL", "http://localhost:9636"), "API URL")
	fs.StringVar(&flags.APIKey, "api-key", getEnv("TRINDEX_API_KEY", ""), "API key")

	if err := fs.Parse(args); err != nil {
		return err
	}

	return runExportWithFlags(ctx, flags, os.Stdout)
}

func runExportWithFlags(ctx context.Context, flags *ExportFlags, defaultOutput io.Writer) error {
	url := flags.APIURL + "/api/export"
	params := "?"
	if flags.Namespace != "" {
		params += "namespace=" + flags.Namespace + "&"
	}
	if flags.Since != "" {
		if _, err := time.Parse(time.RFC3339, flags.Since); err != nil {
			return fmt.Errorf("invalid since date: %w", err)
		}
		params += "since=" + flags.Since + "&"
	}
	if flags.Until != "" {
		if _, err := time.Parse(time.RFC3339, flags.Until); err != nil {
			return fmt.Errorf("invalid until date: %w", err)
		}
		params += "until=" + flags.Until + "&"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url+params, nil)
	if err != nil {
		return err
	}

	if flags.APIKey != "" {
		req.Header.Set("X-API-Key", flags.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	output := defaultOutput
	if flags.Output != "" {
		file, err := os.Create(flags.Output)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer func() { _ = file.Close() }()
		output = file
		fmt.Printf("Exporting to %s...\n", flags.Output)
	}

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if flags.Output == "" {
		_, _ = fmt.Fprintln(defaultOutput)
	}
	_, _ = fmt.Fprintln(defaultOutput, "Export complete")
	return nil
}
