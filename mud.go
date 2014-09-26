package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const PORT = 8888

var world *World
var debugLog, infoLog, errorLog *log.Logger

type World struct {
	players Set
	rooms Set
}

type Room struct {
	name, description string
}

type Player struct {
	name, description string
	conn net.Conn
	location *Room
}

func NewWorld() *World {
	return &World{}
}

func (w *World) NewPlayer(name string, location *Room) (*Player, error) {

	player := &Player{name: name, location: location}

	if w.players.ContainsWhere(func (i interface{}) bool {return true}) {
		return nil, errors.New("User already exists")
	} else {
		w.players.Add(player)
		return player, nil
	}
}

func (w *World) NewRoom(name string) (r *Room, err error) {
	r = &Room{name: name}
	w.rooms.Add(r)
	// No errors, for now
	return
}

//
// Connect a player
//
func (w *World) ConnectPlayer(name string, conn net.Conn) (p *Player, err error) {

	users := w.players.Find(func (i interface{}) bool {
		return i.(Player).name == name
	})

	if len(users) != 1 {
		err = errors.New("User not found")
	} else {
		var pl Player = users[0].(Player)
		pl.conn = conn
	}

	return
}

//
// Disconnect a player
//
func (w *World) DisconnectPlayer(name string) (p *Player, err error) {
	p = w.players[name]

	if p == nil {
		err = errors.New("User not found")
	} else {
		p.conn = nil
	}

	return
}

//
// Send a message to a player
//
func (p *Player) tell(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	p.conn.Write([]byte(s))
}

//
// Handle a single client connection loop
//
func playerLoop(conn net.Conn) {
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

		debugLog.Println("User said:", string(linebuf[:n]))

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

			go playerLoop(conn)
		}
	}()

	<-stopRequested

	// Notify all clients, clean up resources, etc.

	infoLog.Println("Shutdown complete. Goodbye!")
}
