package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"auction-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSMessage định nghĩa cấu trúc tin nhắn chung
type WSMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// Client đại diện cho một kết nối WebSocket
type Client struct {
	UserID    uint
	AuctionID uint
	Conn      *websocket.Conn
	Send      chan WSMessage
}

// Room quản lý danh sách client trong một phiên đấu giá
type Room struct {
	AuctionID uint
	Clients   map[*Client]bool
	Lock      sync.RWMutex
}

// AuctionHub quản lý tất cả các phòng
type AuctionHub struct {
	Rooms map[uint]*Room
	Lock  sync.RWMutex
}

var Hub = &AuctionHub{
	Rooms: make(map[uint]*Room),
}

type WSHandler struct {
	auctionService *services.AuctionService
}

func NewWSHandler(auctionService *services.AuctionService) *WSHandler {
	return &WSHandler{auctionService: auctionService}
}

func (h *WSHandler) ServeWS(c *gin.Context) {
	// 1. Parse Auction ID
	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid auction id"})
		return
	}

	// 2. Xác thực UserID (ưu tiên Context -> Query/Header)
	var userID uint
	if uid, ok := c.Get("user_id"); ok {
		if id, okVal := uid.(uint); okVal {
			userID = id
		}
	}

	// 3. Nâng cấp kết nối
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}

	client := &Client{
		UserID:    userID,
		AuctionID: uint(auctionID),
		Conn:      conn,
		Send:      make(chan WSMessage, 256),
	}

	Hub.register(client)

	// Chạy các goroutine để đọc và ghi
	go client.writePump()
	go h.readPump(client)
}

// --- Hub Logic ---

func (h *AuctionHub) register(c *Client) {
	h.Lock.Lock()
	room, exists := h.Rooms[c.AuctionID]
	if !exists {
		room = &Room{
			AuctionID: c.AuctionID,
			Clients:   make(map[*Client]bool),
		}
		h.Rooms[c.AuctionID] = room
	}
	h.Lock.Unlock()

	room.Lock.Lock()
	room.Clients[c] = true
	count := len(room.Clients)
	room.Lock.Unlock()

	h.Broadcast(c.AuctionID, WSMessage{
		Event:   "viewer_count",
		Payload: gin.H{"count": count},
	})
}

func (h *AuctionHub) unregister(c *Client) {
	h.Lock.RLock()
	room, exists := h.Rooms[c.AuctionID]
	h.Lock.RUnlock()

	if exists {
		room.Lock.Lock()
		if _, ok := room.Clients[c]; ok {
			delete(room.Clients, c)
			close(c.Send)
		}
		count := len(room.Clients)
		room.Lock.Unlock()

		c.Conn.Close()

		h.Broadcast(c.AuctionID, WSMessage{
			Event:   "viewer_count",
			Payload: gin.H{"count": count},
		})
	}
}

func (h *AuctionHub) Broadcast(auctionID uint, msg WSMessage) {
	h.Lock.RLock()
	room, exists := h.Rooms[auctionID]
	h.Lock.RUnlock()

	if !exists {
		return
	}

	room.Lock.RLock()
	defer room.Lock.RUnlock()

	for client := range room.Clients {
		select {
		case client.Send <- msg:
		default:
			// Nếu channel đầy, ngắt kết nối client đó
			close(client.Send)
			delete(room.Clients, client)
		}
	}
}

// --- Pump Logic ---

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteJSON(msg); err != nil {
			break
		}
	}
}

func (h *WSHandler) readPump(c *Client) {
	defer Hub.unregister(c)

	// Cấu trúc nhận dữ liệu linh hoạt
	type IncomingRequest struct {
		Event   string          `json:"event"`
		Payload json.RawMessage `json:"payload"`
	}

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var req IncomingRequest
		if err := json.Unmarshal(message, &req); err != nil {
			continue // Bỏ qua nếu message sai format
		}

		switch req.Event {
		case "place_bid":
			if c.UserID == 0 {
				c.Send <- WSMessage{Event: "error", Payload: "Authentication required"}
				continue
			}

			var bidPayload struct {
				Amount float64 `json:"amount"`
			}
			if err := json.Unmarshal(req.Payload, &bidPayload); err != nil {
				c.Send <- WSMessage{Event: "error", Payload: "Invalid bid payload"}
				continue
			}

			// Gọi Service đã được định nghĩa đầy đủ
			bid, extended, newEndAt, err := h.auctionService.ProcessBid(c.AuctionID, c.UserID, bidPayload.Amount)
			if err != nil {
				c.Send <- WSMessage{Event: "bid_error", Payload: err.Error()}
				continue
			}

			// Broadcast thành công
			Hub.Broadcast(c.AuctionID, WSMessage{
				Event: "new_bid",
				Payload: gin.H{
					"bid_id":     bid.ID,
					"amount":     bid.Amount,
					"bidder_id":  bid.BidderID,
					"bid_type":   bid.BidType,
					"created_at": bid.CreatedAt,
				},
			})

			if extended && newEndAt != nil {
				Hub.Broadcast(c.AuctionID, WSMessage{
					Event: "anti_snipe_extended",
					Payload: gin.H{
						"new_end_at": newEndAt.Format(time.RFC3339),
					},
				})
			}
		}
	}
}