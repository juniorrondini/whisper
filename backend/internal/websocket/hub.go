package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"whisper/backend/internal/auth"
	"whisper/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	clients    map[string]map[*Client]bool
}

type Event struct {
	CompanyID      string `json:"company_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	Type           string `json:"type"`
	Payload        any    `json:"payload"`
}

type Client struct {
	hub       *Hub
	app       *service.App
	conn      *ws.Conn
	send      chan Event
	userID    uuid.UUID
	companyID uuid.UUID
	role      string
}

type inbound struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
	Status         string `json:"status"`
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event, 64),
		clients:    map[string]map[*Client]bool{},
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			key := client.companyID.String()
			if h.clients[key] == nil {
				h.clients[key] = map[*Client]bool{}
			}
			h.clients[key][client] = true
			h.broadcast <- Event{
				CompanyID: key,
				Type:      "presence.online",
				Payload:   gin.H{"user_id": client.userID.String(), "role": client.role},
			}
		case client := <-h.unregister:
			key := client.companyID.String()
			if clients := h.clients[key]; clients != nil {
				if clients[client] {
					delete(clients, client)
					close(client.send)
					h.broadcast <- Event{
						CompanyID: key,
						Type:      "presence.offline",
						Payload:   gin.H{"user_id": client.userID.String(), "role": client.role},
					}
				}
				if len(clients) == 0 {
					delete(h.clients, key)
				}
			}
		case event := <-h.broadcast:
			for client := range h.clients[event.CompanyID] {
				select {
				case client.send <- event:
				default:
					delete(h.clients[event.CompanyID], client)
					close(client.send)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}

func Serve(app *service.App, hub *Hub, jwtSecret string) gin.HandlerFunc {
	upgrader := ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		token := c.Query("token")
		claims, err := auth.ParseAccessToken(jwtSecret, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &Client{
			hub:       hub,
			app:       app,
			conn:      conn,
			send:      make(chan Event, 16),
			userID:    claims.UserID,
			companyID: claims.CompanyID,
			role:      claims.Role,
		}
		hub.register <- client
		go client.writePump()
		go client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(8192)
	_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})
	for {
		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg inbound
		if err := json.Unmarshal(bytes, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "message.send":
			senderID := c.userID
			message, err := c.app.CreateMessage(context.Background(), c.companyID, msg.ConversationID, "agent", &senderID, msg.Content)
			if err != nil {
				c.send <- Event{CompanyID: c.companyID.String(), Type: "message.error", Payload: gin.H{"error": err.Error()}}
				continue
			}
			c.hub.Broadcast(Event{CompanyID: c.companyID.String(), ConversationID: msg.ConversationID, Type: "message.created", Payload: message})
		case "typing.start", "typing.stop":
			c.hub.Broadcast(Event{CompanyID: c.companyID.String(), ConversationID: msg.ConversationID, Type: msg.Type, Payload: gin.H{"user_id": c.userID.String()}})
		case "message.read":
			c.hub.Broadcast(Event{CompanyID: c.companyID.String(), ConversationID: msg.ConversationID, Type: "message.read", Payload: gin.H{"user_id": c.userID.String()}})
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case event, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(ws.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
