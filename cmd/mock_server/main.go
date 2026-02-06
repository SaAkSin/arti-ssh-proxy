package main

import (
	"bufio"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	log.Println("Agent connected!")

	// Send 'date\n' to agent automatically to verify execution
	c.WriteMessage(websocket.BinaryMessage, []byte("date\n"))

	// WS -> Stdout
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			os.Stdout.Write(message)
		}
	}()

	// Stdin -> WS
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		// Send as binary (pty input)
		// Add newline because scanner strips it
		err := c.WriteMessage(websocket.BinaryMessage, []byte(text+"\n"))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {
	log.Println("Starting Mock Server on :8080/ws")
	http.HandleFunc("/ws", echo)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
