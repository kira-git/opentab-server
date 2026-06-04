package main

import (
	"log"
	"net/http"
	"time"

	"opentab-server/internal/config"
	"opentab-server/internal/database"
	"opentab-server/internal/repositories"
	"opentab-server/internal/routes"
	"opentab-server/internal/services"

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
		db, err := database.Connect(cfg.DatabaseURL, database.PoolOptions{
			MaxOpenConns:       cfg.DBMaxOpenConns,
			MaxIdleConns:       cfg.DBMaxIdleConns,
			ConnMaxLifetimeMin: cfg.DBConnMaxLifetimeMin,
		})
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
		log.Printf("OpenTab server database pool configured: maxOpen=%d maxIdle=%d lifetimeMin=%d", cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetimeMin)
	}

	routes.RegisterWithStatusAndOptions(router, repos, runtimeStatus, routes.HandlerOptions{
		OnCall: services.OnCallOptions{
			AIConcurrentLimit:     cfg.AIConcurrentLimit,
			AIUserConcurrentLimit: cfg.AIUserConcurrentLimit,
			SmoothInterval:        cfg.AIStreamSmoothInterval,
			SmoothChunkSize:       cfg.AIStreamSmoothChunkSize,
		},
	})

	address := cfg.Host + ":" + cfg.Port
	log.Printf("OpenTab server mode=%s listening on %s", cfg.AppMode, address)
	log.Printf("OpenTab AI options: globalLimit=%d userLimit=%d smoothInterval=%s smoothChunkSize=%d", cfg.AIConcurrentLimit, cfg.AIUserConcurrentLimit, cfg.AIStreamSmoothInterval, cfg.AIStreamSmoothChunkSize)

	server := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
