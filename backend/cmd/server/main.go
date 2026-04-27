package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"whisper/backend/internal/config"
	"whisper/backend/internal/database"
	"whisper/backend/internal/handler"
	"whisper/backend/internal/repository"
	"whisper/backend/internal/service"
	realtime "whisper/backend/internal/websocket"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db, err := database.ConnectPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	redis, err := database.ConnectRedis(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Printf("redis unavailable, continuing without rate-limit/presence cache: %v", err)
		redis = nil
	}
	if redis != nil {
		defer redis.Close()
	}

	if cfg.AutoMigrate {
		if err := database.Migrate(context.Background(), db, cfg.MigrationsPath); err != nil {
			log.Fatalf("migrations: %v", err)
		}
	}

	app := service.NewApp(cfg, repository.NewStore(db), redis)
	if cfg.SeedDemo {
		if err := app.SeedDemo(context.Background()); err != nil {
			log.Printf("seed skipped: %v", err)
		}
	}

	hub := realtime.NewHub()
	go hub.Run()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.NewRouter(cfg, app, hub),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("%s API listening on %s", cfg.AppName, cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
