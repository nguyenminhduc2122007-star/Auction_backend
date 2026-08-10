package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var baseURL = "http://localhost:8081"

func doRequest(method, path, token string, body interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		// print body for debugging
		fmt.Fprintf(os.Stderr, "raw response: %s\n", string(b))
		return nil, err
	}
	return out, nil
}

func main() {
	// Register seller
	fmt.Println("== Register seller ==")
	sellerReg := map[string]interface{}{"email": "seller@example.com", "password": "password123", "full_name": "Seller Test", "user_type": "Seller"}
	r, err := doRequest("POST", "/api/auth/register", "", sellerReg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(r)

	// Register admin
	fmt.Println("== Register admin ==")
	adminReg := map[string]interface{}{"email": "admin@example.com", "password": "password123", "full_name": "Admin Test", "user_type": "Admin"}
	r, err = doRequest("POST", "/api/auth/register", "", adminReg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(r)

	// Login seller
	fmt.Println("== Login seller ==")
	sellerLogin := map[string]interface{}{"email": "seller@example.com", "password": "password123"}
	lr, err := doRequest("POST", "/api/auth/login", "", sellerLogin)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(lr)
	tokenS := ""
	if data, ok := lr["data"].(map[string]interface{}); ok {
		if t, ok2 := data["token"].(string); ok2 {
			tokenS = t
		}
	}

	// Login admin
	fmt.Println("== Login admin ==")
	adminLogin := map[string]interface{}{"email": "admin@example.com", "password": "password123"}
	la, err := doRequest("POST", "/api/auth/login", "", adminLogin)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(la)
	tokenA := ""
	if data, ok := la["data"].(map[string]interface{}); ok {
		if t, ok2 := data["token"].(string); ok2 {
			tokenA = t
		}
	}

	// Seller create item
	fmt.Println("== Seller create item ==")
	item := map[string]interface{}{"title": "Test Auction 1", "type": "Electronics", "description": "Test item", "price": 100.0, "start_price": 100.0, "current_price": 100.0}
	ci, err := doRequest("POST", "/api/items", tokenS, item)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(ci)
	itemID := int64(0)
	if data, ok := ci["data"].(map[string]interface{}); ok {
		if idf, ok2 := data["id"].(float64); ok2 {
			itemID = int64(idf)
		}
	}

	// Admin list items
	fmt.Println("== Admin list items ==")
	list, err := doRequest("GET", "/api/items", tokenA, nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(list)

	// Admin update status
	if itemID > 0 {
		fmt.Println("== Admin update status ==")
		upd := map[string]interface{}{"status": "active"}
		p := fmt.Sprintf("/api/items/%d/status", itemID)
		uresp, err := doRequest("PUT", p, tokenA, upd)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		fmt.Println(uresp)
	}

	// Final admin list active
	fmt.Println("== Admin list active ==")
	alist, err := doRequest("GET", "/api/items?status=active", tokenA, nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(alist)
}
