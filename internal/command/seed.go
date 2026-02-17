package command

import (
	"github.com/psds-microservice/session-manager-service/internal/database"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	return database.RunSeeds(db)
}
