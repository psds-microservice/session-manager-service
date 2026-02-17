package cmd

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/session-manager-service/internal/command"
	"github.com/psds-microservice/session-manager-service/internal/config"
	"github.com/psds-microservice/session-manager-service/internal/database"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run migrations and seeds",
	RunE:  runSeed,
}

func runSeed(cmd *cobra.Command, args []string) error {
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
	db, err := database.Open(cfg.DSN())
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	if err := command.Seed(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	log.Println("seed: ok")
	return nil
}
