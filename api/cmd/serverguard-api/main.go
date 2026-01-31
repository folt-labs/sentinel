package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/folt-labs/sentinel/api/internal/config"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/handlers"
	"github.com/folt-labs/sentinel/api/internal/middleware"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Sentinel API %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Sentinel API",
		ServerHeader: "Sentinel",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	// Setup middleware
	middleware.Setup(app, cfg)

	// Setup routes
	handlers.Setup(app, db, cfg)

	// Handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		log.Printf("Sentinel API %s listening on %s", version, addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Sentinel API stopped")
}
