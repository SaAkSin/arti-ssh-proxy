package main

import (
	"arti-ssh-agent/internal/pty"
	"arti-ssh-agent/internal/ws"
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

	// 1. Start PTY
	ptySvc := pty.NewService()
	f, err := ptySvc.Start()
	if err != nil {
		log.Fatalf("Failed to start PTY: %v", err)
	}
	defer ptySvc.Close()
	ptySvc.SetDefaultSize()

	log.Println("PTY started successfully")

	// 2. Setup PTY Output Channel
	ptyOutput := make(chan []byte, 1024)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY Read Error: %v", err)
				}
				close(ptyOutput)
				return
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			ptyOutput <- data
		}
	}()

	// 3. Main Connection Loop
	wsClient := ws.NewClient(serverURL)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		log.Println("Connecting to server...")
		if err := wsClient.Connect(); err != nil {
			log.Printf("Connection failed: %v. Retrying in 3s...", err)
			select {
			case <-interrupt:
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
					if err := wsClient.WriteBinary(data); err != nil {
						log.Printf("WS Write Error: %v", err)
						return
					}
				case <-stopWriter:
					return
				}
			}
		}()

		// Blocking: WS -> PTY
		err := wsClient.ReadLoop(
			func(data []byte) {
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
		close(stopWriter) // Stop the writer
		wsClient.Close()
		
		select {
		case <-interrupt:
			return
		case <-time.After(1 * time.Second):
			// continue
		}
	}
}
