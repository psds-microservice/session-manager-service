package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

func RunSeeds(db *gorm.DB) error {
	cwd, _ := os.Getwd()
	dirs := []string{
		filepath.Join(cwd, "database", "seeds"),
		filepath.Join(cwd, "..", "database", "seeds"),
	}
	var seedsDir string
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			seedsDir, _ = filepath.Abs(d)
			break
		}
	}
	if seedsDir == "" {
		return fmt.Errorf("seeds dir not found")
	}
	entries, err := os.ReadDir(seedsDir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, f := range files {
		path := filepath.Join(seedsDir, f)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("seed %s: %w", f, err)
		}
		if err := db.Exec(string(body)).Error; err != nil {
			return fmt.Errorf("seed %s: %w", f, err)
		}
		log.Printf("seed: applied %s", f)
	}
	return nil
}
