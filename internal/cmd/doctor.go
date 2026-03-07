package cmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
)

type DoctorFlags struct {
	RemoteURL string
	APIKey    string
}

func NewDoctorCommand() *Command {
	flags := &DoctorFlags{}
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	fs.StringVar(&flags.RemoteURL, "remote", "", "Remote Trindex HTTP API URL (proxy mode)")
	fs.StringVar(&flags.APIKey, "api-key", "", "API key for remote connection (proxy mode)")

	return &Command{
		Name:        "doctor",
		Description: "Run diagnostics",
		Flags:       fs,
		Run: func(ctx context.Context, args []string) error {
			exitCode := RunDoctor(ctx, flags)
			os.Exit(exitCode)
			return nil
		},
	}
}

func RunDoctor(ctx context.Context, flags *DoctorFlags) int {
	fmt.Println("🔍 Trindex Doctor")
	fmt.Println()

	serverURL := flags.RemoteURL
	if serverURL == "" {
		serverURL = os.Getenv("TRINDEX_URL")
	}

	apiKey := flags.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("TRINDEX_API_KEY")
	}

	if serverURL != "" && serverURL != "local" {
		return RunProxyDoctor(ctx, serverURL, apiKey)
	}

	allPassed := true
	oldLevel := slog.SetLogLoggerLevel(slog.LevelError)
	defer slog.SetLogLoggerLevel(oldLevel)

	fmt.Print("Checking configuration... ")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ FAILED\n   %v\n", err)
		allPassed = false
	} else {
		fmt.Println("✅ PASSED")
		fmt.Printf("   Database URL: %s\n", maskPassword(cfg.DatabaseURL))
		fmt.Printf("   Embed Model: %s\n", cfg.EmbedModel)
		fmt.Printf("   Embed Dimensions: %d\n", cfg.EmbedDimensions)
	}
	fmt.Println()

	if cfg != nil {
		fmt.Print("Checking database connection... ")
		database, err := db.New(cfg)
		if err != nil {
			fmt.Printf("❌ FAILED\n   %v\n", err)
			allPassed = false
		} else {
			if err := database.Health(ctx); err != nil {
				fmt.Printf("❌ FAILED\n   %v\n", err)
				allPassed = false
			} else {
				fmt.Println("✅ PASSED")

				var tableExists bool
				err := database.Pool().QueryRow(ctx,
					"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'memories')").Scan(&tableExists)
				if err != nil {
					fmt.Printf("   ⚠️  Could not check tables: %v\n", err)
				} else if !tableExists {
					fmt.Println("   ⚠️  Database tables not initialized")
					fmt.Println("      Run: trindex server (auto-migrates on startup)")
					fmt.Println("      Or:  trindex mcp (auto-migrates on startup)")
				} else {
					var count int
					err := database.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM memories").Scan(&count)
					if err != nil {
						fmt.Printf("   ⚠️  Could not count memories: %v\n", err)
					} else {
						fmt.Printf("   Memories in database: %d\n", count)
					}
				}
			}
			database.Close()
		}
		fmt.Println()
	}

	if cfg != nil {
		fmt.Print("Checking embedding endpoint... ")
		client := embed.NewClient(cfg)
		dims, err := testEmbedConnection(client)
		if err != nil {
			fmt.Printf("❌ FAILED\n   %v\n", err)
			allPassed = false
		} else {
			fmt.Println("✅ PASSED")
			fmt.Printf("   Endpoint: %s\n", cfg.EmbedBaseURL)
			fmt.Printf("   Returned dimensions: %d\n", dims)
			if dims != cfg.EmbedDimensions {
				fmt.Printf("   ⚠️  WARNING: Configured dimensions (%d) != Actual dimensions (%d)\n",
					cfg.EmbedDimensions, dims)
			}
		}
		fmt.Println()
	}

	if allPassed {
		fmt.Println("🎉 All checks passed! Trindex is ready to go.")
		return 0
	}
	fmt.Println("❌ Some checks failed. Please fix the issues above.")
	return 1
}

func maskPassword(connStr string) string {
	if idx := strings.Index(connStr, "://"); idx != -1 {
		prefix := connStr[:idx+3]
		rest := connStr[idx+3:]
		if atIdx := strings.Index(rest, "@"); atIdx != -1 {
			userPass := rest[:atIdx]
			if colonIdx := strings.LastIndex(userPass, ":"); colonIdx != -1 {
				user := userPass[:colonIdx]
				return prefix + user + ":***@" + rest[atIdx+1:]
			}
		}
	}
	return connStr
}

func testEmbedConnection(client *embed.Client) (int, error) {
	testText := "test"
	embedding, err := client.Embed(testText)
	if err != nil {
		return 0, err
	}
	return len(embedding), nil
}

func RunProxyDoctor(ctx context.Context, serverURL, apiKey string) int {
	fmt.Println("   [Mode: Proxy Client]")
	fmt.Printf("   Checking remote connection to: %s\n", serverURL)
	fmt.Println()

	allPassed := true

	fmt.Print("Checking proxy /health endpoint... ")

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/health", nil)
	if err != nil {
		fmt.Printf("❌ FAILED\n   Failed to create request: %v\n", err)
		allPassed = false
	} else {
		if apiKey != "" {
			req.Header.Set("X-API-Key", apiKey)
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			fmt.Printf("❌ FAILED\n   Connection error: %v\n", err)
			allPassed = false
		} else {
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode == http.StatusOK {
				fmt.Println("✅ PASSED")
			} else {
				fmt.Printf("❌ FAILED\n   HTTP Status: %d\n", resp.StatusCode)
				if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
					fmt.Println("   ⚠️  Check your --api-key or TRINDEX_API_KEY environment variable")
				}
				allPassed = false
			}
		}
	}

	fmt.Println()

	if allPassed {
		fmt.Println("🎉 All proxy checks passed! Trindex MCP is ready to go.")
		return 0
	}
	fmt.Println("❌ Some checks failed. Please fix the issues above.")
	return 1
}
