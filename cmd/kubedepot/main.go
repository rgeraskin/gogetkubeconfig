package main

import (
	"embed"

	"github.com/rgeraskin/kubedepot/internal/config"
	"github.com/rgeraskin/kubedepot/internal/server"
)

//go:embed kodata/web/*
var embeddedFiles embed.FS

func main() {
	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	logger := cfg.Logger
	logger.Info("Starting kubedepot")

	// Log effective configuration
	logger.Info("Configuration loaded",
		"port", cfg.Port,
		"configsDir", cfg.ConfigsDir,
		"webDir", cfg.WebDir,
		"debug", cfg.Debug,
	)

	// Create server configuration
	serverConfig := &server.Server{
		ConfigsDir:    cfg.ConfigsDir,
		WebDir:        cfg.WebDir,
		Logger:        logger,
		EmbeddedFiles: &embeddedFiles,
	}

	// Create and start server
	srv, err := server.NewServer(serverConfig)
	if err != nil {
		logger.Fatalf("Failed to initialize server: %+v", err)
	}

	logger.Debug("Starting server", "port", cfg.Port)
	if err := srv.Start(cfg.Port); err != nil {
		logger.Fatalf("Server failed: %+v", err)
	}
}
