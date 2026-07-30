package main

import (
	"context"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("Expected App instance")
	}
	if len(app.logEntries) != 0 {
		t.Errorf("Expected 0 log entries, got %d", len(app.logEntries))
	}
}

func TestStartupAndBroadcast(t *testing.T) {
	app := NewApp()
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start it up
	app.startup(ctx)
	
	if app.ctx == nil {
		t.Error("Expected context to be set")
	}
	
	// Wait a bit to let broadcastLoop tick at least once
	time.Sleep(100 * time.Millisecond)
	
	// Cancel it to gracefully exit the broadcastLoop
	cancel()
	
	// Wait for goroutine to return
	time.Sleep(50 * time.Millisecond)
}
