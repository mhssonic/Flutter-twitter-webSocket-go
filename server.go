package main

import (
	"Flutter_Go_WebSocket/DAO"
	"Flutter_Go_WebSocket/model"
	"Flutter_Go_WebSocket/tools"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var connections = make(map[int]*websocket.Conn)
var connectionLocker sync.Mutex

func directMessageChatHandler(db *database.DataBase) func(w http.ResponseWriter, r *http.Request) {
	// upgrade this connection to a WebSocket
	// connection
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
		}
		defer func(ws *websocket.Conn) {
			err := ws.Close()
			if err != nil {
				fmt.Println(err)
				return
			}
		}(ws)

		log.Println("Client Connected")
		token := r.Header.Get("Cookie")
		token = strings.Split(token, "token=")[1]
		id, err := tools.JwtDecode(token)
		if err != nil {
			log.Println(err)
			return
		}

		targetId, err := strconv.Atoi(r.Header.Get("target-id"))
		if err != nil {
			log.Println(err)
			return
		}
		messages, err := db.GetMessages(id, targetId)
		if err != nil {
			log.Println(err)
		} else {
			messageJson, err := json.Marshal(messages)
			err = ws.WriteMessage(1, messageJson)
			if err != nil {
				log.Println(err)
			}
		}

		connectionLocker.Lock()
		connections[id] = ws
		connectionLocker.Unlock()
		ws.SetCloseHandler(disconnectWebSocket(id))

		wg.Add(2)
		go reader(ws, id, db)
		wg.Wait()
	}
}

func writer(conn *websocket.Conn, messages []model.DirectMessage) {
	bytes, err := json.Marshal(messages)
	if err != nil {
		log.Println(err)
		return
	}
	if err := conn.WriteMessage(1, bytes); err != nil {
		log.Println(err)
		return
	}
}

func disconnectWebSocket(id int) func(code int, text string) error {
	return func(code int, text string) error {
		connectionLocker.Lock()
		delete(connections, id)
		connectionLocker.Unlock()
		//TODO nil?
		return nil
	}
}

func reader(conn *websocket.Conn, id int, db *database.DataBase) {
	defer wg.Done()
	for {
		// read in a message
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		var message model.DirectMessage
		err = json.Unmarshal(p, &message)
		if err != nil {
			log.Println(err)
			return
		}

		message.AuthorId = id
		err = db.SaveMessage(&message)
		if err != nil {
			log.Println(err)
			return
		}

		messages := make([]model.DirectMessage, 1)
		messages[0] = message
		writer(conn, messages)
		connectionLocker.Lock()
		wsTarget, ok := connections[message.TargetId]
		connectionLocker.Unlock()
		if ok {
			writer(wsTarget, messages)
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var wg = sync.WaitGroup{}

func main() {
	var db database.DataBase
	db.ConnectToDataBases()

	http.HandleFunc("/show-direct", directMessageChatHandler(&db))
	log.Fatal(http.ListenAndServe(":6060", nil))
}
