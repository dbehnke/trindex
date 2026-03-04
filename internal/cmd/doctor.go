package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
)

func NewDoctorCommand() *Command {
	return &Command{
		Name:        "doctor",
		Description: "Run diagnostics",
		Run: func(ctx context.Context, args []string) error {
			exitCode := RunDoctor(ctx)
			os.Exit(exitCode)
			return nil
		},
	}
}

func RunDoctor(ctx context.Context) int {
	fmt.Println("🔍 Trindex Doctor")
	fmt.Println()

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

				var count int
				err := database.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM memories").Scan(&count)
				if err != nil {
					fmt.Printf("   ⚠️  Table check failed: %v\n", err)
				} else {
					fmt.Printf("   Memories in database: %d\n", count)
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
