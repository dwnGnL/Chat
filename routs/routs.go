package routs

import(
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	// "github.com/jinzhu/gorm"
	"time"
	"fmt"
	"kr/models"
	"kr/db"
	"log"
)

var onlineUsers = make(map[*websocket.Conn]string)
var users = make(map[*websocket.Conn]bool)
var broadcast = make(chan models.Message)
var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func Auth(c *gin.Context)  {
	c.HTML(http.StatusOK,"index.html",gin.H{"title":"authorization"})
}
var login string
func CheckLog(c *gin.Context){
	var account []models.Account
	var check bool
	db := db.GetDB()
	login=c.PostForm("login")
	pass:=c.PostForm("password")
	db.Find(&account)
	for _,acc:=range account{
		if login==acc.Login && pass==acc.Pass{
			logs := models.Logs{User:acc.Login}
			db.Create(&logs)
			c.Redirect(303,"http://localhost:8080/chat")
			check=true
		}
	}
	if !check{
		c.Redirect(303,"http://localhost:8080/auth")
	}
}

func Chat(c *gin.Context){
	db := db.GetDB()
	var logs models.Logs
	db.First(&logs)
	if logs.User==""{
		c.HTML(http.StatusOK,"index.html",gin.H{"title":"authorization"})
	}else{
		c.HTML(http.StatusOK,"chat.html",gin.H{"title":"dsdsa"})
	}
	
	
}

func Wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	database:=db.GetDB()
	if err != nil {
		fmt.Printf("Failed to set websocket upgrade: %+v \n", err)
		return
	}
	var logs models.Logs
	database.First(&logs)
	database.Delete(&logs)
	defer conn.Close()

	var history []models.History
	onlineUsers[conn] = logs.User
	users[conn] = true

	database.Find(&history)

	for _, row:= range history{
		historyMsg := models.Message {
			User: row.User,
			Message: row.Message,
			Date: row.Date,
		}
		conn.WriteJSON(historyMsg)
	}

	for {
		var msg models.Message
		// Read in a new message as JSON and map it to a Message object
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(users, conn)
			delete(onlineUsers,conn)
			break
		}
		if msg.Message == "connect" {
			msg.User = onlineUsers[conn]
			msg.Message = "test.conn"
			conn.WriteJSON(msg)
		}else {
			msg.User = onlineUsers[conn]
			// Send the newly received message to the broadcast channel
			broadcast <- msg
		}
	}
}

func HandleMessages() {
	database:=db.GetDB()
	now := time.Now().Format("02.01.2006 15:04:05")
	for {
		var history models.History
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		if msg.Message != " is online" {
			history.User = msg.User
			history.Message = msg.Message
			history.Date = now

			database.Create(&history)
		}

		// Send it out to every user that is currently connected
		for user := range users {
			msg.Date = now
			err := user.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				user.Close()
				delete(users, user)
				delete(onlineUsers,user)
			}
		}
	}
}
