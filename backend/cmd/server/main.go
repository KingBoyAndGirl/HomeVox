package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/KingBoyAndGirl/HomeVox/backend/internal/api"
	"github.com/KingBoyAndGirl/HomeVox/backend/internal/config"
)

func main() {
	cfg := config.Load()
	if err := validateFrontendDir(cfg.FrontendDir); err != nil {
		log.Fatalf("frontend build unavailable: %v", err)
	}
	router, cleanup := api.NewRouterWithCleanup(cfg, cfg.FrontendDir)
	defer cleanup()

	log.Printf("HomeVox listening on %s (API + frontend %s)", cfg.ListenAddr, cfg.FrontendDir)
	if err := router.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func validateFrontendDir(frontendDir string) error {
	indexPath := filepath.Join(frontendDir, "index.html")
	info, err := os.Stat(indexPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", indexPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is not a file", indexPath)
	}
	return nil
}
