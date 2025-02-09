package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type WebSocketServer struct {
	ID        string
	Clients   map[*websocket.Conn]bool
	Broadcast chan *Message
}

type Message struct {
	ClientName string `json:"clientName"`
	Content    string `json:"content"`
}

func (s *WebSocketServer) HandleMessages() {
	for {
		msg := <-s.Broadcast

		// Send the message to all Clients
		for client := range s.Clients {
			err := client.WriteMessage(websocket.TextMessage, getMessageTemplate(msg))
			if err != nil {
				log.Printf("Write  Error: %v ", err)
				client.Close()
				delete(s.Clients, client)
			}
		}
	}
}

func (s *WebSocketServer) HandleWebSocket(ctx *websocket.Conn) {
	// Register a new Client
	s.Clients[ctx] = true
	defer func() {
		delete(s.Clients, ctx)
		ctx.Close()
	}()

	for {
		_, msg, err := ctx.ReadMessage()
		if err != nil {
			log.Println("Read Error:", err)
			break
		}

		// send the message to the broadcast channel
		log.Println(string(msg))
		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Println("Error Unmarshalling:", err) // Use log.Println instead of log.Fatalf to avoid exiting
			continue                                  // Continue to the next message
		}
		message.ClientName = s.ID

		s.Broadcast <- &message
	}
}

func NewWebSocket() *WebSocketServer {
	return &WebSocketServer{
		ID:        uuid.New().String(),
		Clients:   make(map[*websocket.Conn]bool),
		Broadcast: make(chan *Message),
	}
}

func (s *WebSocketServer) HandleConnections(ctx *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(ctx) {
		ctx.Locals("allowed", true)
		return ctx.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (s *WebSocketServer) HandleWebSocketConnection(ctx *websocket.Conn) {
	// moved HandleWebSocket logic here
	s.HandleWebSocket(ctx)
}

func (s *WebSocketServer) ProcessMessages() {
	// Renamed to HandleMessages and moved logic here
	s.HandleMessages()
}

func getMessageTemplate(msg *Message) []byte {
	tmpl, err := template.ParseFiles("views/message.html")
	if err != nil {
		log.Fatalf("template parsing: %s", err)
	}

	// Render the template with the message as data.
	var renderedMessage bytes.Buffer
	err = tmpl.Execute(&renderedMessage, msg)
	if err != nil {
		log.Fatalf("template execution: %s", err)
	}

	return renderedMessage.Bytes()
}

func main() {
	app := fiber.New()
	server := NewWebSocket()

	// Serve static files (if needed)
	app.Static("/", "./public")

	// WebSocket route
	app.Use("/ws", server.HandleConnections)
	app.Get("/ws", websocket.New(server.HandleWebSocketConnection))

	// Start processing messages in a separate goroutine
	go server.ProcessMessages()

	log.Fatal(app.Listen(":3000"))
}
