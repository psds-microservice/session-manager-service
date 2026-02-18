package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/session-manager-service/internal/config"
	"github.com/psds-microservice/session-manager-service/internal/database"
	"github.com/psds-microservice/session-manager-service/internal/kafka"
	"github.com/psds-microservice/session-manager-service/internal/model"
	"github.com/psds-microservice/session-manager-service/internal/searchindex"
	"github.com/spf13/cobra"
)

var reindexSearchCmd = &cobra.Command{
	Use:   "reindex-search",
	Short: "Reindex all consultation sessions into search (Elasticsearch). Uses SEARCH_SERVICE_URL if set, otherwise skips (normal indexing via Kafka).",
	RunE:  runReindexSearch,
}

func init() {
	rootCmd.AddCommand(reindexSearchCmd)
}

func runReindexSearch(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	db, err := database.Open(cfg.DSN())
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	var sessions []model.ConsultationSession
	if err := db.Find(&sessions).Error; err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	log.Printf("reindex-search: found %d sessions", len(sessions))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Приоритет: Kafka > HTTP
	if len(cfg.KafkaBrokers) > 0 && cfg.KafkaTopicSession != "" {
		log.Println("reindex-search: using Kafka for reindexing")
		producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopicSession)
		defer producer.Close()
		for i := range sessions {
			producer.ProduceSessionEvent(ctx, "session.created", sessions[i].ID, map[string]interface{}{
				"client_id": sessions[i].ClientID.String(),
				"pin":       sessions[i].PIN,
				"status":    sessions[i].Status,
			})
			if (i+1)%50 == 0 || i == len(sessions)-1 {
				log.Printf("reindex-search: sent %d/%d events to Kafka", i+1, len(sessions))
			}
		}
		log.Printf("reindex-search: done, sent %d events to Kafka (search-service worker will index them)", len(sessions))
		return nil
	}

	if cfg.SearchServiceURL != "" {
		log.Println("reindex-search: using HTTP for reindexing")
		client := searchindex.NewClient(cfg.SearchServiceURL)
		for i := range sessions {
			client.IndexSession(ctx, &sessions[i])
			if (i+1)%50 == 0 || i == len(sessions)-1 {
				log.Printf("reindex-search: indexed %d/%d", i+1, len(sessions))
			}
		}
		log.Printf("reindex-search: done, indexed %d sessions via HTTP", len(sessions))
		return nil
	}

	log.Println("reindex-search: neither KAFKA_BROKERS nor SEARCH_SERVICE_URL set")
	log.Println("reindex-search: normal indexing happens via Kafka events (search-service worker)")
	log.Printf("reindex-search: found %d sessions (not reindexed)", len(sessions))
	return nil
}
