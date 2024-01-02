package config

import (
	"errors"
	"log"
	"os"
	"path"

	"github.com/blang/semver"
)

// migrations is a map of version to migration function.
// Warning: RunMigration() runs the migrations in arbitrary order.
var migrations = map[string]func() error{
	"v0.3.0": func() error {
		errStr := Config.AddFilterList(filterList{
			Name:    "DandelionSprout's URL Shortener",
			Type:    "privacy",
			Url:     "https://raw.githubusercontent.com/DandelionSprout/adfilt/master/LegitimateURLShortener.txt",
			Enabled: true,
		})
		if errStr != "" {
			err := errors.New(errStr)
			return err
		}
		return nil
	},
}

// RunMigrations runs the version-to-version migrations.
func RunMigrations() {
	if Version == "development" {
		log.Println("skipping migrations in development mode")
		return
	}

	var lastMigration string
	lastMigrationFile := path.Join(Config.ConfigDir, "last_migration")
	if firstLaunch {
		lastMigration = Version
	} else {
		if _, err := os.Stat(lastMigrationFile); !os.IsNotExist(err) {
			lastMigrationData, err := os.ReadFile(lastMigrationFile)
			if err != nil {
				log.Fatalf("failed to read last migration file: %v", err)
			}
			lastMigration = string(lastMigrationData)
		} else {
			// Should trigger when updating from pre v0.3.0
			lastMigration = "v0.0.0"
		}
	}

	lastMigrationV, err := semver.ParseTolerant(lastMigration)
	if err != nil {
		log.Printf("error parsing last migration(%s): %v\n", lastMigration, err)
		return
	}

	for version, migration := range migrations {
		versionV, err := semver.ParseTolerant(version)
		if err != nil {
			log.Printf("error parsing migration version(%s): %v\n", version, err)
			continue
		}

		if lastMigrationV.LT(versionV) {
			if err := migration(); err != nil {
				log.Printf("error running migration(%s): %v\n", version, err)
			} else {
				log.Printf("ran migration %s\n", version)
			}
		}
	}

	if err := os.WriteFile(lastMigrationFile, []byte(Version), 0644); err != nil {
		log.Printf("error writing last migration file: %v\n", err)
	}
}
