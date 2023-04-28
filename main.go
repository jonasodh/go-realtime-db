package main

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Message struct {
	Action string `json:"action"`
	Data   Data   `json:"data"`
}

type Data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var db *sql.DB

func updateData(db *sql.DB, key, value string) error {
	query := `INSERT INTO my_table (my_key, my_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE my_value = ?`

	result, err := db.Exec(query, key, value, value)
	if err != nil {
		return fmt.Errorf("error updating data: %v", err)
	}

	// Check if any rows were affected.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows were updated")
	}

	return nil
}

func handleWebSocketConnection(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	allowedOrigins := []string{"http://localhost:3000"}
	origin := r.Header.Get("Origin")
	isAllowed := false
	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		http.Error(w, "Origin not allowed", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})

	if err != nil {
		log.Println("WebSocket accept error:", err)
		return
	}

	defer func(conn *websocket.Conn, code websocket.StatusCode, reason string) {
		err := conn.Close(code, reason)
		if err != nil {
			log.Println("Error closing connection:", err)
		}
	}(conn, websocket.StatusInternalError, "Closing connection")

	for {
		var message Message
		err = wsjson.Read(r.Context(), conn, &message)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				log.Println("Client closed the connection")
			} else {
				log.Println("Read error:", err)
			}
			break
		}

		log.Printf("Received message: %v", message)

		switch message.Action {
		case "update":
			err = updateData(db, message.Data.Key, message.Data.Value)
			if err != nil {
				log.Println("Error updating data:", err)
				response := map[string]interface{}{
					"status":  "error",
					"message": "Failed to update data",
				}
				err = wsjson.Write(r.Context(), conn, response)
				if err != nil {
					log.Println("Write error:", err)
					break
				}
			} else {
				response := map[string]interface{}{
					"status":  "success",
					"message": "Data updated successfully",
				}
				err = wsjson.Write(r.Context(), conn, response)
				if err != nil {
					log.Println("Write error:", err)
					break
				}
			}
		default:
			log.Println("Unknown action:", message.Action)
			response := map[string]interface{}{
				"status":  "error",
				"message": "Unknown action",
			}
			err = wsjson.Write(r.Context(), conn, response)
			if err != nil {
				log.Println("Write error:", err)
				break
			}
		}
	}

	err = conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		return
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db, err = ConnectToMySQL()
	if err != nil {
		log.Fatal("Error connecting to MySQL:", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal("Error closing MySQL connection:", err)
		}
	}(db)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocketConnection(w, r, db)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
