package command

import (
	"github.com/psds-microservice/session-manager-service/internal/database"
)

func MigrateUp(databaseURL string) error {
	return database.MigrateUp(databaseURL)
}
