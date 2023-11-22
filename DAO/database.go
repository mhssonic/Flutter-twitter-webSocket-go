package database

import (
	"Flutter_Go_WebSocket/model"
	"Flutter_Go_WebSocket/tools"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	_ "go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"time"
)

type DataBase struct {
	pq         *sql.DB
	collection *mongo.Collection
}

func (db *DataBase) ConnectToDataBases() {
	db.pq = connectToDataBasePostqresql()
	client := connectToDataBaseMongodb()
	databaseMongodb := client.Database("flutter")
	db.collection = databaseMongodb.Collection("direct_message")
}

func connectToDataBaseMongodb() *mongo.Client {
	uri := "mongodb://localhost:27017"
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func connectToDataBasePostqresql() *sql.DB {
	const (
		host     = "localhost"
		port     = 5432
		user     = "postgres"
		password = "pashmak"
		dbname   = "flutter"
	)

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	// open database
	var err error
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		log.Printf("server: connecting to databse: %s\n", err)
	}
	return db
}

func (db *DataBase) SaveMessage(message *model.DirectMessage) error {
	//todo check validation
	var err error
	message.MessageId, err = getNextDirectMessageId(db.pq)
	if err != nil {
		return err
	}
	message.PostingTime = time.Now()
	_, err = db.collection.InsertOne(context.Background(), message)
	if err != nil {
		return err
	}
	chatBoxId := tools.JenkinsHash(message.AuthorId, message.TargetId, true)
	hasChatBox, err := containChatBox(db.pq, chatBoxId)
	if err != nil {
		return err
	}
	if !hasChatBox {
		creatChatBox(db.pq, chatBoxId)
		userAddToFriend(db.pq, message.AuthorId, message.TargetId)
	}
	appendMessageToChatBox(db.pq, chatBoxId, message.MessageId)
	return nil
}

func (db *DataBase) GetMessages(id int, targetId int) ([]model.DirectMessage, error) {
	//TODO check validation
	messagesId, err := getMessageIds(db.pq, tools.JenkinsHash(id, targetId, true))
	if err != nil {
		return nil, err
	}
	return getMessagesByIds(db.collection, messagesId)
}

func getNextDirectMessageId(db *sql.DB) (int, error) {
	exec, err := db.Query(`select NEXTVAL('seq_message_id')`)
	if err != nil {
		return 0, err
	}
	if exec.Next() {
		var messageId int
		err := exec.Scan(&messageId)
		if err != nil {
			return 0, err
		}
		return messageId, nil
	}
	return 0, errors.New("pq couldn't get next id")
}

func containFieldKey(db *sql.DB, table string, field string, value any) (bool, error) {
	query, err := db.Query("SELECT * FROM "+table+" WHERE "+field+" = $1", value)
	if err != nil {
		log.Println(err)
		return false, err
	}
	return query.Next(), nil
}

func containChatBox(db *sql.DB, id int) (bool, error) {
	return containFieldKey(db, "chat_box", "id", id)
}

func creatChatBox(db *sql.DB, id int) {
	_, err := db.Exec(`insert into chat_box ("id") values($1)`, id)
	if err != nil {
		log.Println(err)
		return
	}
}

func userAddToFriend(db *sql.DB, user int, userTarget int) {
	appendToArrayField(db, "users", user, "friend", userTarget)
	appendToArrayField(db, "users", userTarget, "friend", user)
}

func appendToArrayField(db *sql.DB, table string, id int, field string, value any) {
	_, err := db.Exec("UPDATE "+table+" SET "+field+" = array_append("+field+",$1) WHERE id = $2", value, id)
	if err != nil {
		log.Println(err)
		return
	}
}

func appendMessageToChatBox(db *sql.DB, chatBox int, messageId int) {
	appendToArrayField(db, "chat_box", chatBox, "message_id", messageId)
}

func getMessageIds(db *sql.DB, chatBoxId int) ([]int32, error) {
	query, err := db.Query(`SELECT * FROM chat_box WHERE id = $1`, chatBoxId)
	if err != nil {
		return nil, err
	}
	if query.Next() {
		var id int
		var messageIds []int32
		err := query.Scan(&id, (*pq.Int32Array)(&messageIds))
		if err != nil {
			return nil, err
		}
		return messageIds, nil
	}
	return nil, errors.New("pq couldn't get message ids from database")
}

func getMessagesByIds(collection *mongo.Collection, messageIds []int32) ([]model.DirectMessage, error) {
	messages := make([]model.DirectMessage, len(messageIds))
	var err error
	for i := 0; i < len(messageIds); i++ {
		messages[i], err = getMessageById(collection, int(messageIds[i]))
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

func getMessageById(collection *mongo.Collection, id int) (model.DirectMessage, error) {
	var directMessage model.DirectMessage
	err := collection.FindOne(context.Background(), bson.D{{"_id", id}}).Decode(&directMessage)
	if err != nil {
		return model.DirectMessage{}, err
	}
	return directMessage, nil
}
