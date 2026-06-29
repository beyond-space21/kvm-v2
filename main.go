package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kvm_v2/Events"
	authhandler "kvm_v2/Events/auth"
	"kvm_v2/internal/repository"
	"kvm_v2/services/auth"
	"kvm_v2/services/config"
	"kvm_v2/services/db"
)

func main() {
	cfg := config.Load()

	if cfg.RunMigrations {
		if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Println("migrations applied")
	}

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	if err := authhandler.BootstrapAdmin(context.Background(), repository.NewUserRepository(database), cfg.BootstrapEmail, cfg.BootstrapPass); err != nil {
		log.Printf("bootstrap admin: %v", err)
	}

	authService, err := auth.NewService(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	router := events.NewRouter(events.Dependencies{
		DB:   database,
		Auth: authService,
		Cfg:  cfg,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
