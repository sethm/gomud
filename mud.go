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
var connections Set
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
type Client struct {
	conn net.Conn
	player *Player
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

func (c *Client) tell(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	c.conn.Write([]byte(s))
}

type Exit struct {
	key int
	name, description string
	destination *Room
}

// Exit implements Object interface
func (e Exit) Key() int { return e.key }
func (e Exit) Name() string { return e.name }
func (e Exit) Description() string { return e.description }

type Room struct {
	key int
	name, description string
	exits Set
}

func (r *Room) NewExit(name string, destination *Room) (e *Exit, err error) {
	foundExit := r.exits.ContainsWhere(func (o Object) bool {
		return o.Name() == name
	})

	if foundExit {
		err = errors.New("An exit with that name already exists.")
	} else {
		e = &Exit{name: name, destination: destination}
		r.exits.Add(e)
	}

	return
}

// Room implements Object interface
func (r Room) Key() int { return r.key }
func (r Room) Name() string { return r.name }
func (r Room) Description() string { return r.description }

type Player struct {
	key int
	name, description string
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
	r = &Room{key: idGen(), name: name, exits: NewSet()}
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

//
// Handle a single client connection loop
//
func connectionLoop(conn net.Conn) {
	linebuf := make([]byte, 1024, 1024)
	client := NewClient(conn)

	// Loop on input and handle it.
	for {
		client.tell("mud> ")
		n, err := conn.Read(linebuf)

		if err != nil {
			if err != io.EOF {
				errorLog.Println("Error:", err)
			}
			break
		}

		line := strings.TrimSpace(string(linebuf[:n]))
		debugLog.Println(fmt.Sprintf("[%s]: %s", conn.RemoteAddr(), line))
		client.tell("Huh?\r\n")
	}

	infoLog.Println("Disconnection from", conn.RemoteAddr())
	conn.Close()
}

//
// Build up the world.
//
func initWorld() {
	connections = NewSet()
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
