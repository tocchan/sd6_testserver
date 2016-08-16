package main

import (
   "encoding/hex"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net"
	"net/http"
	"strconv"
)

const gAddress string = ""
const gTCPPort string = "4325"
const gUDPPort string = "6325"

const gDebugPort string = "3443"

const CONN_TCP int = 0
const CONN_UCP int = 1

const MSG_JOIN int = 0
const MSG_LEAVE int = 1
const MSG_DATA int = 2

var gServerConn net.Conn

// What needs to be sent
// Addr String
// Size
// Content
type DebugMessage struct {
   conn_type int `json:"conn_type"`
	msg_type int `json:"msg_type"`
	addr     string `json:"addr"`
	data     string `json:"data"`
}

type DebugRoom struct {
	messages    chan *DebugMessage
	incoming    chan *websocket.Conn
   outgoing    chan *websocket.Conn
	connections []*websocket.Conn
}

//----------------------------------------------------------------------------------
func (room *DebugRoom) SendMessage(msg *DebugMessage) {
	for i := 0; i < len(room.connections); i += 1 {
		conn := room.connections[0]
		err := websocket.JSON.Send(conn, msg)
      if (err != nil) {
         room.outgoing <- conn
      }
	}
}

//----------------------------------------------------------------------------------
func (room *DebugRoom) RemoveConnection(ws *websocket.Conn) {
   for i,c := range room.connections {
      if (c == ws) {
         count := len(room.connections)
         room.connections[i] = room.connections[count - 1]
         room.connections = room.connections[:count -1];
         return;
      }
   }
}

//----------------------------------------------------------------------------------
func (room *DebugRoom) DoWork() {
	for {
		select {
		case conn := <-room.incoming:
			room.connections = append(room.connections, conn)
         fmt.Println( "New debug connection: " + conn.LocalAddr().String() )
      case conn := <- room.outgoing:
         fmt.Println( "Removing debug connection: " + conn.LocalAddr().String() )
         room.RemoveConnection(conn)
		case msg := <-room.messages:
			room.SendMessage(msg)
		}
	}
}

//----------------------------------------------------------------------------------
func DebugServerWork(msgs chan *DebugMessage) {
	// Debug Server listens for connections

	// Kick off the room
	room := DebugRoom{
		messages:    msgs,
		incoming:    make(chan *websocket.Conn),
      outgoing:    make(chan *websocket.Conn),
		connections: make([]*websocket.Conn, 0),
	}
	go room.DoWork()

	// Esstablish the WebSocket listener
	on_connect := func(ws *websocket.Conn) {
		room.incoming <- ws

      AddMessage(msgs, CONN_TCP, MSG_JOIN, nil, nil, 0)

      // If this function leaves - the connection closes I think
      buffer := make([]byte, 512)
      for {

         _, err := ws.Read(buffer)
         if (err != nil) {
            break;
         }
      }

      room.outgoing <- ws
      ws.Close()
	}

	http.Handle("/debug", websocket.Handler(on_connect))
	log.Fatal(http.ListenAndServe("localhost:" + gDebugPort, nil))

	fmt.Println("Debug Server Listening on: localhost:" + gDebugPort)

	for {
		// ever
	}
}

//----------------------------------------------------------------------------------
func AddMessage(msgs chan *DebugMessage, ctype int, mtype int, conn net.Conn, data []byte, data_size int) {
	msg := DebugMessage{
		msg_type: mtype,
      conn_type: ctype,
		addr:     "invalid",
		data:     "",
	}

   if (conn != nil) {
      msg.addr = conn.LocalAddr().String()
   }

	if data != nil {
      buffer := make([]byte,2048)
		n := hex.Encode(buffer, data)
      msg.data = string(buffer[:n])
	}

	msgs <- &msg
}

//----------------------------------------------------------------------------------
func TCPClientWork(conn net.Conn, msgs chan *DebugMessage) {
	AddMessage(msgs, CONN_TCP, MSG_JOIN, conn, nil, 0)

	fmt.Println("Received new client connection at: " + conn.LocalAddr().String())
	buffer := make([]byte, 2045)

	for {
		count, err := conn.Read(buffer)
		if (err != nil) && (count > 0) {
			fmt.Println("Received " + strconv.Itoa(count) + "B from " + conn.LocalAddr().String())
			AddMessage(msgs, CONN_TCP, MSG_DATA, conn, buffer, count)
		} else {
			break
		}
	}

	AddMessage(msgs, CONN_TCP, MSG_LEAVE, conn, nil, 0)
}

//----------------------------------------------------------------------------------
func TCPServerWork(msgs chan *DebugMessage) {
	// Listens for new connections.
	ln, err := net.Listen("tcp", gAddress+":"+gTCPPort)

	if err != nil {
		log.Fatal("TCPServer: ", err)
		return
	}

	fmt.Println("TCP Server Listening on: " + ln.Addr().String())

	// Create a new thread for the client
	for {
		conn, err := ln.Accept()
		if (err != nil) && (conn != nil) {
			go TCPClientWork(conn, msgs)
		}
	}

	ln.Close()
}

//----------------------------------------------------------------------------------
func main() {

	debug_channel := make(chan *DebugMessage)

	go DebugServerWork(debug_channel)
   go TCPServerWork(debug_channel)

	for {
		// infinite loop
	}
}
