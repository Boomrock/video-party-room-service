package room

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	CommandPlay        CommandType = "play"
	CommandPause       CommandType = "pause"
	CommandSeek        CommandType = "seek"
	CommandSync        CommandType = "sync"
	CommandError       CommandType = "error"
	CommandVideoChange CommandType = "change-video"

	pongWait   = 30 * time.Second
	pingPeriod = 25 * time.Second
)

type CommandType string

type Message struct {
	Type      CommandType `json:"type"`
	From      *Client     `json:"-"` // не сериализуется
	Time      float64     `json:"time,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	payload   string      `json:"payload"`
}

type Client struct {
	Conn *websocket.Conn
	Room *Room
	send chan *Message

	mu sync.Mutex // для защиты от повторного close
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.send:
	default:
		close(c.send)
	}
	_ = c.Conn.Close()
}

func (c *Client) run() {
	go c.receiveHandler()
	go c.sendHandler()
}

func (c *Client) receiveHandler() {
	defer func() {
		c.Room.unregister <- c
		c.close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				slog.Warn("read error", "error", err, "client", c.Conn.RemoteAddr())
			}
			return
		}

		msg := &Message{}
		if err := json.Unmarshal(data, msg); err != nil {
			slog.Warn("invalid JSON", "error", err, "data", string(data))
			continue
		}

		// Валидация типа команды
		switch msg.Type {
		case CommandPlay, CommandPause, CommandSeek, CommandSync, CommandVideoChange:
			// OK
		default:
			slog.Warn("unknown command type", "type", msg.Type)
			continue
		}

		msg.From = c
		msg.Timestamp = time.Now()
		c.Room.message <- msg
	}
}

func (c *Client) sendHandler() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Канал закрыт — клиент отключён
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				slog.Error("failed to marshal message", "error", err)
				continue
			}

			c.Conn.SetWriteDeadline(time.Now().Add(pingPeriod))
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(pingPeriod))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Room struct {
	register   chan *Client
	unregister chan *Client
	message    chan *Message
	clients    map[*Client]bool
	mx         sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	cleaned atomic.Bool
}

func NewRoom() *Room {
	ctx, cancel := context.WithCancel(context.Background())
	room := &Room{
		ctx:        ctx,
		cancel:     cancel,
		register:   make(chan *Client),
		unregister: make(chan *Client, 10), // буферизован, чтобы избежать блокировок
		message:    make(chan *Message, 10),
		clients:    make(map[*Client]bool),
	}
	return room
}

func (r *Room) ClientCount() int {
	r.mx.RLock()
	defer r.mx.RUnlock()
	return len(r.clients)
}

func (r *Room) cleanup() {
	if !r.cleaned.CompareAndSwap(false, true) {
		return // уже очищено
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	count := len(r.clients)
	for client := range r.clients {
		close(client.send)
		client.Conn.Close()
	}
	r.clients = nil

	slog.Info("Room closed", "client_count", count)
}

func (r *Room) registerClient(client *Client) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.clients[client] = true
	client.run()
}

func (r *Room) unregisterClient(client *Client) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if _, ok := r.clients[client]; ok {
		delete(r.clients, client)
		client.close()

		if len(r.clients) == 0 {
			r.cancel()
		}
	}
}

func (r *Room) sendMessage(message *Message) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	for client := range r.clients {
		if client == message.From {
			continue
		}
		select {
		case client.send <- message:
		default:
			client.close()
		}
	}
}

func (r *Room) Run() {
	go func() {
		for {
			select {
			case client := <-r.register:
				r.registerClient(client)
			case client := <-r.unregister:
				r.unregisterClient(client)
			case message := <-r.message:
				r.sendMessage(message)
			case <-r.ctx.Done():
				r.cleanup()
				return
			}
		}
	}()
}

// Hub управляет комнатами
type Hub struct {
	Rooms map[string]*Room
	mx    sync.RWMutex
}

func (h *Hub) getRoom(key string) *Room {
	if key == "" {
		return nil
	}

	h.mx.RLock()
	room := h.Rooms[key]
	h.mx.RUnlock()

	if room != nil {
		return room
	}

	h.mx.Lock()
	defer h.mx.Unlock()

	// Double-check
	if room, exists := h.Rooms[key]; exists {
		return room
	}

	room = NewRoom()
	h.Rooms[key] = room
	room.Run()

	// Автоудаление из хаба при завершении
	go func() {
		<-room.ctx.Done()
		h.mx.Lock()
		delete(h.Rooms, key)
		h.mx.Unlock()
		slog.Info("Room removed from hub", "key", key)
	}()

	return room
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// В продакшене ограничьте домены!
		return true
	},
}

var hub = Hub{
	Rooms: make(map[string]*Room),
}

func VideoController() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "missing 'key' query parameter", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("WebSocket upgrade failed",
				"remote_addr", r.RemoteAddr,
				"method", r.Method,
				"path", r.URL.Path,
				"error", err,
			)
			http.Error(w, "Failed to upgrade connection", http.StatusBadRequest)
			return
		}

		room := hub.getRoom(key)
		if room == nil {
			http.Error(w, "Invalid room", http.StatusNotFound)
			return
		}

		client := &Client{
			Conn: conn,
			Room: room,
			send: make(chan *Message, 10),
		}

		room.register <- client
		slog.Info("Client connected",
			"room_key", key,
			"client_ip", r.RemoteAddr,
			"total_clients", room.ClientCount(),
		)
	}
}
