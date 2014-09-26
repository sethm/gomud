package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const PORT = 8888

var world *World
var debugLog, infoLog, errorLog *log.Logger
var idGen func() int = KeyGen()

//
// Generate unique IDs for objects
//

func KeyGen() func() int {
	c := 0
	return func() int {
		c += 1
		return c
	}
}

// Wrapper for net.Conn with some convenience functions
type Connection struct {
	conn net.Conn
}

func (c *Connection) tell(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	c.conn.Write([]byte(s))
}

type Room struct {
	key int
	name, description string
}

// Room implements Object interface
func (r Room) Key() int { return r.key }
func (r Room) Name() string { return r.name }
func (r Room) Description() string { return r.description }

type Exit struct {
	key int
	name, description string
}

// Exit implements Object interface
func (e Exit) Key() int { return e.key }
func (e Exit) Name() string { return e.name }
func (e Exit) Description() string { return e.description }

type Player struct {
	key int
	name, description string
	conn *Connection
	location *Room
}

// Player implements Object interface
func (p Player) Key() int { return p.key }
func (p Player) Name() string { return p.name }
func (p Player) Description() string { return p.description }

type World struct {
	players Set
	rooms Set
	exits Set
}

func NewWorld() *World {
	return &World{NewSet(), NewSet(), NewSet()}
}

func (w *World) NewRoom(name string) (r *Room, err error) {
	r = &Room{key: idGen(), name: name}
	w.rooms.Add(r)

	return
}

func (w *World) NewPlayer(name string, location *Room) (p *Player, err error) {
	if w.players.ContainsWhere(func (o Object) bool {return o.Name() == name}) {
		err = errors.New("User already exists")
	} else {
		p = &Player{key: idGen(), name: name, location: location}
		w.players.Add(p)
	}

	return
}

func (w *World) ConnectPlayer(name string, conn net.Conn) (p *Player, err error) {

	player := w.players.SelectFirst(func (o Object) bool {
		return o.Name() == name
	})

	if player != nil {
		p = player.(*Player)
		p.conn = &Connection{conn}
	} else {
		err = errors.New("Player not found")
	}

	return 
}

func (w *World) DisconnectPlayer(name string) (p *Player, err error) {
	player := w.players.SelectFirst(func (o Object) bool {
		return o.Name() == name
	})

	if player != nil {
		p = player.(*Player)
		p.conn = nil
	} else {
		err = errors.New("User not found")
	}

	return
}

//
// Send a message to a player
//
func (p *Player) tell(msg string, args ...interface{}) {
	if p.conn != nil {
		p.conn.tell(msg, args...)
	}
}

//
// Handle a single client connection loop
//
func connectionLoop(conn net.Conn) {
	linebuf := make([]byte, 1024, 1024)

	// Loop on input and handle it.
	for {
		conn.Write([]byte("mud> "))

		n, err := conn.Read(linebuf)

		if err != nil {
			if err != io.EOF {
				errorLog.Println("Error:", err)
			}
			break
		}

		line := strings.TrimSpace(string(linebuf[:n]))

		debugLog.Println("User said:", line)

		conn.Write([]byte("Huh?\r\n"))
	}

	infoLog.Println("Disconnection from", conn.RemoteAddr())
	conn.Close()
}

//
// Build up the world.
//
func initWorld() {
	world = NewWorld()
}

//
// Main entry point
//
func main() {
	debugLog = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLog = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	stopRequested := make(chan bool)

	go func() {
		<-sigs
		infoLog.Println("SIGTERM received.")
		stopRequested<- true
	}()

	infoLog.Println("Starting server...")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))

	if err != nil {
		errorLog.Println("Could not start server:", err)
		return
	}

	infoLog.Println("Server listening on port", PORT)

	initWorld()

	go func() {
		for {
			conn, err := ln.Accept()
		
			infoLog.Println("Accepted connection from:", conn.RemoteAddr())

			if err != nil {
				errorLog.Println("Could not accept connection:", err)
				continue
			}

			go connectionLoop(conn)
		}
	}()

	<-stopRequested

	// Notify all clients, clean up resources, etc.

	infoLog.Println("Shutdown complete. Goodbye!")
}
