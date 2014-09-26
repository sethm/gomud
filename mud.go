package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const PORT = 8888

var world *World

type World struct {
	players map[string]*Player
}

type Player struct {
	name, description string
	conn net.Conn
}

func NewWorld() *World {
	return &World{players: make(map[string]*Player)}
}

func (w *World) NewPlayer(name string) (*Player, error) {
	player := &Player{name: name}
	if w.players[name] != nil {
		return nil, errors.New("User already exists")
	} else {
		w.players[name] = player
		return player, nil
	}
}

func (w *World) ConnectPlayer(name string, conn net.Conn) (*Player, error) {
	player := w.players[name]

	if player == nil {
		return player, errors.New("User not found")
	} else {
		player.conn = conn
		return player, nil
	}
}

func (w *World) DisconnectPlayer(name string) (p *Player, err error) {
	p = w.players[name]
	err = nil

	if p != nil {
		p.conn = nil
	} else {
		err = errors.New("User not found")
	}

	return
}

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
				fmt.Println("[ERR] Error:", err)
			}
			break
		}

		fmt.Print("[DBG] User said:", string(linebuf[:n]))

		conn.Write([]byte("Huh?\r\n"))
	}

	fmt.Println("[INFO] Disconnection from", conn.RemoteAddr())
	conn.Close()
}

func initWorld() {
	world = NewWorld()
}

//
// Main entry point
//
func main() {

	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	stopRequested := make(chan bool)

	go func() {
		<-sigs
		fmt.Println("[INFO] SIGTERM received.")
		stopRequested<- true
	}()

	fmt.Println("[INFO] Starting server...")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))

	if err != nil {
		fmt.Println("[ERR] Could not start server:", err)
		return
	}

	fmt.Println("[INFO] Server listening on port", PORT)

	initWorld()

	go func() {
		for {
			conn, err := ln.Accept()
		
			fmt.Println("[INFO] Accepted connection from:", conn.RemoteAddr())

			if err != nil {
				fmt.Println("[ERR] Could not accept connection:", err)
				continue
			}

			go playerLoop(conn)
		}
	}()

	<-stopRequested

	// Notify all clients, clean up resources, etc.

	fmt.Println("[INFO] Shutdown complete. Goodbye!")
}
