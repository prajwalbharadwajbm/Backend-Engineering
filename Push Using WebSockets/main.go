package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	conn     *websocket.Conn
	username string
}

// Chat Message
type Message struct {
	Content  string `json:"content"`
	Username string `json:"username"`
}

var (
	// clients holds all connected clients as map
	// just to make sure we don't push to disconnected clients and crash the server.
	clients = make(map[*Client]bool)

	// broadcast channel for messages
	broadcast = make(chan Message)

	// mutex for thread-safe operations on clients map
	clientsMutex sync.Mutex
)

// upgrader handles WebSocket connection upgrades
// Initially, all connections start as regular HTTP connections
// WebSocket connections begin with an HTTP handshake
// The upgrader handles this handshake process automatically
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// Allowing all origins for my learning
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Check if this is a WebSocket request
	if websocket.IsWebSocketUpgrade(r) {
		// Handle WebSocket connection
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err)
			return
		}
		defer ws.Close()

		// Get the remote address (includes IP and port)
		remoteAddr := ws.RemoteAddr().String()
		// Extract just the port number
		_, port, err := net.SplitHostPort(remoteAddr)
		if err != nil {
			log.Printf("Error getting port: %v", err)
			return
		}

		// Create a new client with port as username
		client := &Client{
			conn:     ws,
			username: fmt.Sprintf("Port%s", port),
		}

		// Add client to the connected clients map
		clientsMutex.Lock()
		clients[client] = true
		clientsMutex.Unlock()

		// Send the username to the client immediately after connection
		welcomeMsg := Message{
			Content:  "Welcome to the chat!",
			Username: client.username,
		}
		err = ws.WriteJSON(welcomeMsg)
		if err != nil {
			log.Printf("Error sending welcome message: %v", err)
			return
		}

		// Remove client when they disconnect
		defer func() {
			clientsMutex.Lock()
			delete(clients, client)
			clientsMutex.Unlock()
			client.conn.Close()
		}()

		for {
			var msg Message
			// Read message from WebSocket
			err := ws.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Error reading message: %v", err)
				}
				break
			}
			// Set the username from the client
			msg.Username = client.username
			// Send the message to the broadcast channel
			broadcast <- msg
		}
	} else {
		// Handle regular HTTP request
		http.Error(w, "Expected WebSocket connection", http.StatusBadRequest)
	}
}

func handleMessages() {
	for {
		// Get the next message from the broadcast channel
		msg := <-broadcast

		// Send message to all connected clients
		clientsMutex.Lock()
		for client := range clients {
			err := client.conn.WriteJSON(msg)
			if err != nil {
				log.Printf("Error writing message: %v", err)
				client.conn.Close()
				delete(clients, client)
				break
			}
		}
		clientsMutex.Unlock()
	}
}

func main() {
	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// Handle WebSocket connections
	http.HandleFunc("/ws", handleConnections)

	// Start the message handler
	go handleMessages()

	// Start the server
	log.Println("Server starting on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
