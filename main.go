package main

import (
	"database/sql"
	"github.com/joho/godotenv"
	"log"
	"net/http"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
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
		var message interface{}
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

		err = wsjson.Write(r.Context(), conn, message)
		if err != nil {
			log.Println("Write error:", err)
			break
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
	db, err := ConnectToMySQL()
	if err != nil {
		log.Fatal("Error connecting to MySQL:", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal("Error closing MySQL connection:", err)
		}
	}(db)

	http.HandleFunc("/ws", handleWebSocketConnection)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
