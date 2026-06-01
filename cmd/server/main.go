package main

import (
	"log"

	"opentab-server/internal/config"
	"opentab-server/internal/database"
	"opentab-server/internal/repositories"
	"opentab-server/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	router := gin.Default()
	repos := repositories.NewMemoryRepositorySet()
	runtimeStatus := routes.RuntimeStatus{
		AppMode:          cfg.AppMode,
		DatabaseEnabled:  false,
		DatabaseType:     "memory",
		AIServiceBaseURL: cfg.AIServiceBaseURL,
	}
	if cfg.DatabaseURL == "" {
		log.Print("OpenTab server DATABASE_URL is empty")
	} else {
		log.Print("OpenTab server DATABASE_URL is configured")
		runtimeStatus.DatabaseEnabled = true
		runtimeStatus.DatabaseType = "postgres"
		db, err := database.Connect(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("OpenTab server database connect failed: %v", err)
		}
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("OpenTab server database migrate failed: %v", err)
		}
		if err := database.Seed(db); err != nil {
			log.Fatalf("OpenTab server database seed failed: %v", err)
		}
		log.Print("OpenTab server database connected, migrated and seeded")
		if cfg.AppMode == "postgres" {
			repos = repositories.NewPostgresRepositorySet(db)
			runtimeStatus.AppMode = "postgres"
			log.Print("OpenTab server repositories switched to PostgreSQL")
		} else {
			log.Print("OpenTab server repositories remain in memory mode")
		}
	}

	routes.RegisterWithStatus(router, repos, runtimeStatus)

	address := cfg.Host + ":" + cfg.Port
	log.Printf("OpenTab server mode=%s listening on %s", cfg.AppMode, address)

	if err := router.Run(address); err != nil {
		log.Fatal(err)
	}
}
