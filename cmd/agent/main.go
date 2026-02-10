package main

import (
	"arti-ssh-agent/internal/pty"
	"arti-ssh-agent/internal/ws"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	var serverURL string
	
	// 1. Flag (Highest Priority)
	flag.StringVar(&serverURL, "url", "", "WebSocket Server URL (overrides env)")
	flag.Parse()

	// 2. Env (Middle Priority)
	if serverURL == "" {
		serverURL = os.Getenv("ARTI_SSH_URL")
	}

	// 3. Default (Lowest Priority)
	if serverURL == "" {
		serverURL = "ws://localhost:8080/ws"
	}

	// 4. Normalize URL Scheme
	if !strings.HasPrefix(serverURL, "ws://") && !strings.HasPrefix(serverURL, "wss://") {
		if strings.Contains(serverURL, "localhost") || strings.Contains(serverURL, "127.0.0.1") {
			serverURL = "ws://" + serverURL
		} else {
			serverURL = "wss://" + serverURL
		}
	}

	log.Printf("Starting Agent. Target: %s", serverURL)

	// Setup Context for Graceful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Start PTY
	ptySvc := pty.NewService()
	f, err := ptySvc.Start()
	if err != nil {
		log.Fatalf("Failed to start PTY: %v", err)
	}
	// Ensure PTY is closed on exit
	defer func() {
		log.Println("Closing PTY...")
		ptySvc.Close()
	}()
	ptySvc.SetDefaultSize()

	log.Println("PTY started successfully")

	// 2. Setup PTY Output Channel
	ptyOutput := make(chan []byte, 1024)
	go func() {
		defer close(ptyOutput)
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if err != nil {
				if err != io.EOF && !errors.Is(err, os.ErrClosed) { // Ignore error if file is closed
					log.Printf("PTY Read Error: %v", err)
				}
				return
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			
			select {
			case ptyOutput <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 3. Main Connection Loop
	wsClient := ws.NewClient(serverURL)
	defer wsClient.Close() // Ensure WS is closed on exit

	// Loop until context is cancelled
	for {
		// Check if we should exit before connecting
		if ctx.Err() != nil {
			log.Println("Context cancelled, exiting main loop")
			return
		}

		log.Println("Connecting to server...")
		if err := wsClient.Connect(); err != nil {
			log.Printf("Connection failed: %v. Retrying in 3s...", err)
			select {
			case <-ctx.Done():
				log.Println("Received signal, stopping reconnection...")
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}

		log.Println("Connected to server")

		// Signal to stop the writer goroutine
		stopWriter := make(chan struct{})

		// Goroutine: PTY -> WS
		go func() {
			defer func() {
				log.Println("Writer routine exited")
			}()
			for {
				select {
				case data, ok := <-ptyOutput:
					if !ok {
						return // PTY closed
					}
					if err := wsClient.WriteData(data); err != nil {
						log.Printf("WS Write Error: %v", err)
						// Close connection to trigger ReadLoop exit and reconnection
						wsClient.Close() 
						return
					}
				case <-stopWriter:
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		// Blocking: WS -> PTY
		// ReadLoop will return error when wsClient.Close() is called (on shutdown or error)
		err := wsClient.ReadLoop(
			func(data []byte) {
				log.Printf("Received data from client: %s", string(data))
				if _, err := f.Write(data); err != nil {
					log.Printf("PTY Write Error: %v", err)
				}
			},
			func(rows, cols uint16) {
				log.Printf("Resizing to %dx%d", rows, cols)
				ptySvc.Resize(rows, cols)
			},
		)
		
		log.Printf("Disconnected: %v", err)
		
		// Cleanup current session
		close(stopWriter)
		wsClient.Close()

		// Check if shutdown was requested
		if ctx.Err() != nil {
			log.Println("Shutting down agent...")
			return
		}

		// Wait a bit before reconnecting
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			// continue
		}
	}
}
