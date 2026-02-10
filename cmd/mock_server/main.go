package main

import (
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

	// Send 'date\n' to agent automatically as Text Message
	c.WriteMessage(websocket.TextMessage, []byte("date\n"))

	// WS -> Stdout
	go func() {
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("Recv Type: %d, Payload: %s", mt, string(message))
			os.Stdout.Write(message)
		}
	}()

	// Keep alive
	select {}
}

func main() {
	log.Println("Starting Mock Server on :8081/ws")
	http.HandleFunc("/ws", echo)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
