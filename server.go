package main

import (
	"bytes"
	"encoding/binary"
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

const MSG_JOIN int = 0
const MSG_LEAVE int = 1
const MSG_DATA int = 2


// What needs to be sent
// Addr String
// Size
// Content
type DebugMessage struct {
	msg_type int
	addr string
	data []byte
}

type DebugRoom struct {
	messages chan *DebugMessage
	incoming chan *websocket.Conn
	connections []*websocket.Conn
}

//----------------------------------------------------------------------------------
func (room *DebugRoom) SendMessage( msg *DebugMessage ) {
	for i := 0; i < len(room.connections); i += 1 {
		conn := room.connections[0]
		websocket.JSON.Write(msg);
	}
}

//----------------------------------------------------------------------------------
func (room *DebugRoom) DoWork() {
	for {
		select {
			case conn := <- room.incoming:
				room.connections = append(room.connections, conn)
			case msg := <- room.messages:
				room.SendMessage( msg )
		}
	}
}

//----------------------------------------------------------------------------------
func DebugServerWork(msgs chan *DebugMessage) {
	// Debug Server listens for connections

	// Kick off the room
	room := DebugRoom { 
		messages: msgs,
		incoming: make(chan net.Conn),
		connections: make([]net.Conn,0),
	}
	go room.DoWork()

	// Esstablish the WebSocket listener
	on_connect := func(ws *websocket.Conn) {
		room.incoming <- ws;
	}

	http.Handle("/debug", websocket.Handler(onConnected))
	http.ListenAndServe(gDebugPort, nill)

	fmt.Println( "Debug Server Listening on: localhost:" + gDebugPort )

	for {
		// ever
	}
}

//----------------------------------------------------------------------------------
func AddMessage( msgs chan *DebugMessage, mtype int, conn net.Conn, data []byte, data_size int ) {
	msg := DebugMessage { 
		msg_type: mtype, 
		addr: conn.LocalAddr().String(),
		data: nil,
	}

	if (data != nil) {
		msg.data = data[:data_size]
	}

	msgs <- &msg
}

//----------------------------------------------------------------------------------
func TCPClientWork( conn net.Conn, msgs chan *DebugMessage) {
	AddMessage( msgs, MSG_JOIN, conn, nil, 0 );

	fmt.Println("Received new client connection at: " + conn.LocalAddr().String())
	buffer := make([]byte, 2045)

	for {
		count, err := conn.Read(buffer)
		if ((err != nil) && (count > 0)) {
			fmt.Println("Received " + strconv.Itoa(count) + "B from " + conn.LocalAddr().String())
			AddMessage( msgs, MSG_DATA, conn, buffer, count )
		} else {
			break
		}
	}

	AddMessage( msgs, MSG_LEAVE, conn, nil, 0 )
}

//----------------------------------------------------------------------------------
func TCPServerWork(msgs chan *DebugMessage) {
	// Listens for new connections.
	ln, err := net.Listen("tcp", gAddress + ":" + gTCPPort)

	if (err != nil) {
		log.Fatal("TCPServer: ", err)
		return
	}

	fmt.Println("TCP Server Listening on: " + ln.Addr().String())

	// Create a new thread for the client
	for {
		conn, err := ln.Accept()
		if ((err != nil) && (conn != nil)) {
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

