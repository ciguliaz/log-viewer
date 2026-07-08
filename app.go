package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/nxadm/tail"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Connection struct {
	App      string `json:"app"`
	Hash     string `json:"hash"`
	Dest     string `json:"dest"`
	Packets  int    `json:"packets"`
	Route    string `json:"route"`
	LastSeen string `json:"last_seen"` // timestamp string
}

// App struct
type App struct {
	ctx         context.Context
	connections map[string]*Connection
	mu          sync.Mutex
	logPath     string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		connections: make(map[string]*Connection),
		logPath:     `C:\Program Files (x86)\hatacone\logs\shadow.log`,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.tailLogs()
	go a.broadcastLoop()
}

var logRegex = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(.+?)\]\s+(.*?)\s+hash=([a-zA-Z0-9]+)\s+:(\d+)\s+→\s+(.*?)\s+→\s+(.*)$`)

func (a *App) tailLogs() {
	if _, err := os.Stat(a.logPath); os.IsNotExist(err) {
		fmt.Println("Log file does not exist:", a.logPath)
	}

	t, err := tail.TailFile(a.logPath, tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // Tail from end
		Logger:    tail.DiscardingLogger,
	})
	if err != nil {
		fmt.Println("Error tailing log:", err)
		return
	}

	for line := range t.Lines {
		a.parseLine(line.Text)
	}
}

func (a *App) parseLine(text string) {
	matches := logRegex.FindStringSubmatch(text)
	if len(matches) < 8 {
		return // Ignore lines that don't match our connection format
	}

	timestamp := matches[1]
	// tag := matches[2]
	appName := matches[3]
	hash := matches[4]
	// srcPort := matches[5]
	dest := matches[6]
	route := matches[7]

	a.mu.Lock()
	defer a.mu.Unlock()

	conn, exists := a.connections[hash]
	if !exists {
		conn = &Connection{
			App:     appName,
			Hash:    hash,
			Dest:    dest,
			Route:   route,
			Packets: 0,
		}
		a.connections[hash] = conn
	}
	conn.Packets++
	conn.LastSeen = timestamp
}

func (a *App) broadcastLoop() {
	ticker := time.NewTicker(500 * time.Millisecond) // Update UI 2 times per second
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			// Convert map to slice for frontend
			var list []Connection
			for _, c := range a.connections {
				list = append(list, *c)
			}
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "connections_update", list)
		}
	}
}

// GetInitialConnections allows frontend to fetch immediately on load
func (a *App) GetInitialConnections() []Connection {
	a.mu.Lock()
	defer a.mu.Unlock()
	var list []Connection
	for _, c := range a.connections {
		list = append(list, *c)
	}
	return list
}
