package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/creativeprojects/go-selfupdate"
)

// AppVersion is injected at build time via -ldflags.
// When running `wails dev` or locally without ldflags, it remains "dev".
var AppVersion = "dev"

// UpdateInfo is returned to the frontend
type UpdateInfo struct {
	Available    bool   `json:"available"`
	Version      string `json:"version"`
	ReleaseNotes string `json:"releaseNotes"`
}

// CheckForUpdate calls the GitHub API to see if a newer version is available.
func (a *App) CheckForUpdate() (UpdateInfo, error) {
	if AppVersion == "dev" {
		// In development mode, we can skip update checks to avoid annoyance,
		// or we can test it by manually setting AppVersion to "v0.0.0".
		log.Println("Skipping update check in dev mode")
		return UpdateInfo{Available: false}, nil
	}

	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug("ciguliaz/log-viewer"))
	if err != nil {
		log.Printf("Error detecting latest version: %v\n", err)
		return UpdateInfo{}, fmt.Errorf("error detecting latest version: %w", err)
	}

	if !found {
		return UpdateInfo{Available: false}, nil
	}

	if latest.LessOrEqual(AppVersion) {
		return UpdateInfo{Available: false}, nil
	}

	log.Printf("Update available: %s -> %s\n", AppVersion, latest.Version())

	return UpdateInfo{
		Available:    true,
		Version:      latest.Version(),
		ReleaseNotes: latest.ReleaseNotes,
	}, nil
}

// ApplyUpdate downloads the new binary, applies it, and returns nil on success.
func (a *App) ApplyUpdate() error {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug("ciguliaz/log-viewer"))
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no update found")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate executable path: %w", err)
	}

	log.Printf("Downloading and applying update %s...\n", latest.Version())
	err = selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe)
	if err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}

	log.Println("Successfully updated to version", latest.Version())
	return nil
}
