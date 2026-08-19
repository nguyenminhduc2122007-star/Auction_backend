package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"auction-backend/internal/services"
	"auction-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// Thời gian chờ ghi tin nhắn vào socket
	writeWait = 10 * time.Second

	// Thời gian chờ nhận tin nhắn/ping từ client
	pongWait = 60 * time.Second

	// Tần suất gửi ping đến client (phải nhỏ hơn pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Kích thước buffer tối đa cho tin nhắn
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Cho phép tất cả các Origin (CORS) kết nối
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSMessage định nghĩa cấu trúc tin nhắn chuẩn gửi/nhận qua WebSocket
type WSMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// Client đại diện cho 1 kết nối WebSocket của 1 người dùng trong 1 phòng
type Client struct {
	UserID    uint
	AuctionID uint
	Conn      *websocket.Conn
	Send      chan WSMessage
	Hub       *AuctionHub
}

// Room quản lý các client đang tham gia xem cùng 1 phiên đấu giá
type Room struct {
	AuctionID uint
	Clients   map[*Client]bool
	Lock      sync.RWMutex
}

// AuctionHub quản lý tất cả các phòng đấu giá trên toàn hệ thống
type AuctionHub struct {
	Rooms map[uint]*Room
	Lock  sync.RWMutex
}

// Global Single Instance Hub - Được gọi chung bởi auction_handler và ws_handler
var Hub = &AuctionHub{
	Rooms: make(map[uint]*Room),
}

// 🟢 1. CÁC HÀM QUẢN LÝ ROOM & BROADCAST OF HUB

// Broadcast gửi tin nhắn tới TẤT CẢ client trong một phòng đấu giá cụ thể
func (h *AuctionHub) Broadcast(auctionID uint, msg WSMessage) {
	h.Lock.RLock()
	room, exists := h.Rooms[auctionID]
	h.Lock.RUnlock()

	if !exists || room == nil {
		return
	}

	room.Lock.RLock()
	defer room.Lock.RUnlock()

	for client := range room.Clients {
		select {
		case client.Send <- msg:
		default:
			// Trường hợp channel bị đầy, đóng kết nối để tránh nghẽn
			close(client.Send)
			delete(room.Clients, client)
		}
	}
}

// Register thêm client vào phòng
func (h *AuctionHub) Register(c *Client) {
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
	room.Lock.Unlock()

	log.Printf("[WebSocket] Client (User: %d) đã tham gia Room Auction #%d", c.UserID, c.AuctionID)
}

// Unregister xóa client khỏi phòng khi ngắt kết nối (Đã fix Race Condition)
func (h *AuctionHub) Unregister(c *Client) {
	h.Lock.Lock()
	defer h.Lock.Unlock()

	room, exists := h.Rooms[c.AuctionID]
	if !exists || room == nil {
		return
	}

	room.Lock.Lock()
	if _, ok := room.Clients[c]; ok {
		delete(room.Clients, c)
		close(c.Send)
	}
	isEmpty := len(room.Clients) == 0
	room.Lock.Unlock()

	// Nếu phòng không còn ai, dọn dẹp room khỏi memory an toàn
	if isEmpty {
		delete(h.Rooms, c.AuctionID)
	}

	log.Printf("[WebSocket] Client (User: %d) đã rời Room Auction #%d", c.UserID, c.AuctionID)
}

// 🟢 2. PUMP LUỒNG ĐỌC/GHI CỦA CLIENT

// ReadPump lắng nghe tin nhắn gửi từ Frontend
func (c *Client) ReadPump(auctionService *services.AuctionService) {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	type IncomingRequest struct {
		Event   string          `json:"event"`
		Payload json.RawMessage `json:"payload"`
	}

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket Error] Read err: %v", err)
			}
			break
		}

		var req IncomingRequest
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		// Xử lý các Event gửi từ Client lên ("place_bid" hoặc "bid")
		switch req.Event {
		case "place_bid", "bid":
			if c.UserID == 0 {
				c.Send <- WSMessage{Event: "bid_error", Payload: "Bạn chưa đăng nhập hoặc Token không hợp lệ"}
				continue
			}

			var bidPayload struct {
				Amount float64 `json:"amount"`
			}
			if err := json.Unmarshal(req.Payload, &bidPayload); err != nil || bidPayload.Amount <= 0 {
				c.Send <- WSMessage{Event: "bid_error", Payload: "Số tiền đặt giá không hợp lệ"}
				continue
			}

			// Gọi ProcessBid trả về 4 giá trị: (bid, extended, newEndAt, err)
			bid, extended, newEndAt, err := auctionService.ProcessBid(c.AuctionID, c.UserID, bidPayload.Amount)
			if err != nil {
				c.Send <- WSMessage{Event: "bid_error", Payload: err.Error()}
				continue
			}

			// Broadcast sự kiện bid:placed tới TẤT CẢ người xem trong room
			Hub.Broadcast(c.AuctionID, WSMessage{
				Event: "bid:placed",
				Payload: gin.H{
					"id":         bid.ID,
					"auction_id": bid.AuctionID,
					"bidder_id":  bid.BidderID,
					"amount":     bid.Amount,
					"bid_type":   bid.BidType,
					"created_at": bid.CreatedAt,
				},
			})

			// Nếu kích hoạt Anti-Snipe gia hạn thời gian
			if extended && newEndAt != nil {
				Hub.Broadcast(c.AuctionID, WSMessage{
					Event: "auction:time_extended",
					Payload: gin.H{
						"auction_id": c.AuctionID,
						"new_end_at": newEndAt,
					},
				})
			}
		}
	}
}

// WritePump gửi tin nhắn từ Hub về cho Trình duyệt Client
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub đã đóng channel
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// 🟢 3. HANDLER NỐI HTTP UPGRADE SANG WEBSOCKET

type WSHandler struct {
	auctionService *services.AuctionService
}

func NewWSHandler(auctionService *services.AuctionService) *WSHandler {
	return &WSHandler{auctionService: auctionService}
}

// ServeWS API Endpoint: GET /api/auctions/:id/ws?token=xyz
func (h *WSHandler) ServeWS(c *gin.Context) {
	// 1. Lấy auction_id từ Route Param hoặc Query String
	auctionIDStr := c.Param("id")
	if auctionIDStr == "" {
		auctionIDStr = c.Query("auction_id")
	}

	auctionID, err := strconv.ParseUint(auctionIDStr, 10, 64)
	if err != nil || auctionID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Auction ID không hợp lệ"})
		return
	}

	// 2. Lấy Token từ Query String hoặc Cookie
	tokenStr := c.Query("token")
	if tokenStr == "" {
		tokenStr, _ = c.Cookie("token")
	}

	var userID uint = 0
	if tokenStr != "" {
		if claims, err := utils.ValidateToken(tokenStr); err == nil {
			userID = claims.UserID
		}
	}

	// 3. Upgrade HTTP Connection sang WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket Error] Upgrade failed: %v", err)
		return
	}

	// 4. Khởi tạo Client và Đăng ký vào Hub
	client := &Client{
		UserID:    userID,
		AuctionID: uint(auctionID),
		Conn:      conn,
		Send:      make(chan WSMessage, 256),
		Hub:       Hub,
	}

	Hub.Register(client)

	// Bắt đầu 2 luồng đọc và ghi chạy song song
	go client.WritePump()
	go client.ReadPump(h.auctionService)
}

// HandleWS Alias đảm bảo tương thích ngược nếu gọi theo tên cũ
func (h *WSHandler) HandleWS(c *gin.Context) {
	h.ServeWS(c)
}
