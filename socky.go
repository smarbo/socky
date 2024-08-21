package socky

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	From    string          `json:"from"`
}

// EventHandler is a handler for socket message related events, example: when the frontend calls 'send_message' to the server
type EventHandler func(event Event, c *Client) error
// ConnectionHandler is a handler for connection related events, example: when the socket is connected or disconnected
type ConnectionHandler func(c *Client) error
// ClientList is a list of Clients
type ClientList map[*Client]bool

// Client is the backend's representation of a socket client
type Client struct {
  // connection is the WebSocket connection.
	connection *websocket.Conn
  // manager is the pointer to the socket manager
  // which contains all clients allowing for broadcasts
	manager    *Manager
  // egress is the channel of events which recieves 
  // events and allows for sending events
	egress     chan Event
  // room is the room name of which the socket is currently connected to
	room       string
  // id is a unique id for the client
  // uses google UUID generator package
  id string
}

func newClient(conn *websocket.Conn, manager *Manager) *Client {
  newID := uuid.New()
	return &Client{
		conn,
		manager,
		make(chan Event),
		"default",
    newID.String(),
	}
}

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

// SendEvent pushes an Event into the egress channel of the Client.
func (c *Client) SendEvent(event Event) {
  c.egress <- event;
}

func (c *Client) BroadcastEvent(event Event) {
  for wsclient := range c.manager.clients {
    wsclient.egress <- event;
  }
} 

func (c *Client) RoomcastEvent(event Event) {
  for wsclient := range c.manager.clients {
    if wsclient.room == c.room {
      wsclient.egress <- event;
    }
  }
}

func (c *Client) readMessages() {
	defer func() {
		// cleanup
		c.manager.RemoveClient(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
	}

	c.connection.SetReadLimit(512)
	c.connection.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("error marshalling event: %v", err)
			break
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println("error handling message: ", err)
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.RemoveClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					fmt.Println("CLOSE conn: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			log.Println("msg")

		case <-ticker.C:
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("writemsg error: ", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Manager struct {
	clients ClientList
	mu sync.RWMutex

  OnConnect ConnectionHandler
  OnDisconnect ConnectionHandler
	handlers map[string]EventHandler
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("no such event type")
	}
}

// Serve is the HTTP server function for socky,
// Usage: http.HandleFunc("/ws", m.Serve)
// where 'm' is your Socky manager instance.
// Serve is not to be called directly, rather
// to be used as a callback for HTTP request handling.
func (m *Manager) Serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	client := newClient(conn, m)
	m.AddClient(client)

  if m.OnConnect != nil {
    m.OnConnect(client)
  }

	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) AddClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client] = true
}

func (m *Manager) RemoveClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[client]; ok {
    if m.OnDisconnect != nil {
      m.OnDisconnect(client)
    }
		client.connection.Close()
		delete(m.clients, client)
	}
}


func newManager() *Manager {
	m := &Manager{
		clients:  make(ClientList),
		handlers: make(map[string]EventHandler),
    OnConnect: nil,
    OnDisconnect: nil,
	}

	m.defaultEventHandlers()
	return m
}



/* Send message event example for a chat room scenario
func SendMessage(event Event, c *Client) error {
	newEvent := Event{
		Type:    "new_message",
		Payload: event.Payload,
		From:    event.From,
	}
	for wsclient := range c.manager.clients {
		if wsclient.room == c.room {
			wsclient.egress <- newEvent
		}
	}
	return nil
}
*/

func ChangeRoom(event Event, c *Client) error {
	c.room = string(event.Payload)
  c.egress <- Event{
    Type: "set_room",
    Payload: event.Payload,
    From: event.From,
  }
	return nil
}

func (m *Manager) defaultEventHandlers() {
  m.AddEventHandler(EventChangeRoom, ChangeRoom)
}

const (
	EventChangeRoom  = "change_room"
)

func (m *Manager) AddEventHandler(msgType string, handler EventHandler) { // Adds an event handler of message type 'msgType' and handler function 'handler'
  m.handlers[msgType] = handler;
}

func Socky() *Manager {
  return newManager()
}
