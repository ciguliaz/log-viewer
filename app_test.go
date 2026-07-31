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
}

func TestStartupAndBroadcast(t *testing.T) {
	app := NewApp()
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start it up
	app.startup(ctx)
	
	if app.ctx == nil {
		t.Error("Expected context to be set")
	}
	
	if app.tailer == nil {
		t.Error("Expected tailer to be instantiated")
	}
	
	// Wait a bit to let broadcastLoop tick at least once
	time.Sleep(100 * time.Millisecond)
	
	// Cancel it to gracefully exit the broadcastLoop
	cancel()
	
	// Wait for goroutine to return
	time.Sleep(50 * time.Millisecond)
}
