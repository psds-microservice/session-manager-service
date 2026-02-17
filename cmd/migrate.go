package cmd

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/session-manager-service/internal/command"
	"github.com/psds-microservice/session-manager-service/internal/config"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE:  runMigrateUp,
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := command.MigrateUp(cfg.DatabaseURL()); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Println("migrate up: ok")
	return nil
}
