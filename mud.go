package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const PORT = 8888

var world *World = NewWorld()
var debugLog, infoLog, errorLog *log.Logger
var idGen func() int = KeyGen()

type CommandHandler func(*World, *Client, Command)

type Flags int

const (
	God Flags = iota
	Wizard
	Builder
	Programmer
	Locked
	TeleportOK
	BuildOK
)

type CommandDesc struct {
	argCount int
	preAuth  bool
	postAuth bool
	handler  CommandHandler
}

type HandlerMap map[string]CommandDesc

var commandHandlers = HandlerMap{
	"@desc":     {2, false, true, doDesc},
	"@dig":      {2, false, true, doDig},
	"@help":     {0, false, true, doHelp},
	"@link":     {2, false, true, doLink},
	"connect":   {2, true, false, doConnect},
	"newplayer": {2, true, false, doNewplayer},
	"emote":     {1, false, true, doEmote},
	"go":        {1, false, true, doMove},
	"help":      {0, false, true, doHelp},
	"look":      {2, false, true, doLook},
	"l":         {2, false, true, doLook},
	"move":      {1, false, true, doMove},
	"quit":      {0, true, true, doQuit},
	"say":       {1, false, true, doSay},
	"tell":      {2, false, true, doTell},
	"walk":      {1, false, true, doMove},
}

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

// A command entered at the MUD's prompt
type Command struct {
	verb   string
	target string
	args   string
}

//
// Abstraction for a client connection that allows us to tie a user
// and a connection together.
//
type Client struct {
	sync.RWMutex
	conn          net.Conn
	player        *Player
	quitRequested bool
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn, quitRequested: false}
}

func (c *Client) Tell(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg+"\r\n", args...)
	c.conn.Write([]byte(s))
}

// TODO: I feel like this needs improvement.
func parseCommand(client *Client, line string) Command {
	// The user may have typed `"foo`, which we want to interpret
	// as "say foo".
	if strings.HasPrefix(line, "\"") {
		return Command{verb: "say", args: line[1:len(line)]}
	}

	if strings.HasPrefix(line, ":") {
		return Command{verb: "emote", args: line[1:len(line)]}
	}

	tokenized := strings.SplitN(line, " ", 2)

	verb := tokenized[0]
	info, isKeyword := commandHandlers[verb]

	// See if we should treat the verb as an exit direction
	if !isKeyword && client.player != nil {
		location := client.player.location
		for _, exit := range location.exits {
			if tokenized[0] == exit.name {
				return Command{verb: "move", args: tokenized[0]}
			}
		}
	}

	if len(tokenized) == 1 {
		return Command{verb: verb}
	}

	if info.argCount == 2 {
		// Further tokenize the args into target/args
		argTokens := strings.SplitN(tokenized[1], " ", 2)

		if len(argTokens) == 1 {
			return Command{verb: verb, target: argTokens[0]}
		}

		return Command{verb: verb, target: argTokens[0], args: argTokens[1]}
	}

	return Command{verb: verb, args: tokenized[1]}
}

func welcome(client *Client) {
	client.Tell("-----------------------------------------------------")
	client.Tell("Welcome to this experimental MUD!")
	client.Tell("")
	client.Tell("To create a new player: newplayer <name> <password>")
	client.Tell("To connect as a player: connect <name> <password>")
	client.Tell("To leave the game:      quit")
	client.Tell("-----------------------------------------------------")
	client.Tell("")
	client.Tell("")
}

//
// Handle a single client connection loop
//
func connectionLoop(conn net.Conn) {
	linebuf := make([]byte, 1024, 1024)
	client := NewClient(conn)

	welcome(client)

	// Loop on input and handle it.
	for {
		// // Uncomment if we want a prompt...
		// client.Tell("mud> ")
		n, err := conn.Read(linebuf)

		if err != nil {
			if err != io.EOF {
				errorLog.Println("Error:", err)
			}
			break
		}

		line := strings.TrimSpace(string(linebuf[:n]))

		if len(line) > 0 {
			command := parseCommand(client, line)
			debugLog.Println("Parsed command:", command)
			if command.verb != "" {
				world.handleCommand(&commandHandlers, client, command)
			}
		}

		if client.quitRequested {
			break
		}
	}

	infoLog.Println("Disconnection from", conn.RemoteAddr())

	if client.player != nil {
		world.TellAllButMe(client.player, client.player.name+" has disconnected.")
		client.player.awake = false
		client.player.client = nil
		client.player = nil
	}

	conn.Close()
}

//
// Build up the world.
//
func initWorld() {
	hall, _ := world.NewRoom("Hallway")
	den, _ := world.NewRoom("The Den")
	kitchen, _ := world.NewRoom("The Kitchen")

	world.NewExit(hall, "east", den)
	world.NewExit(den, "west", hall)
	world.NewExit(den, "south", kitchen)
	world.NewExit(kitchen, "north", den)

	world.NewPlayer("God", "xyzzy", hall)
	world.NewPlayer("Wizard", "xyzzy", hall)
}

func init() {
	debugLog = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLog = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

//
// Main entry point
//
func main() {
	// Set up the SIGTERM signal handler
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	stopRequested := make(chan bool)

	go func() {
		<-sigs
		infoLog.Println("SIGTERM received.")
		stopRequested <- true
	}()

	infoLog.Println("Loading world...")

	initWorld()

	infoLog.Println("World initialized with",
		len(world.rooms), "room(s),",
		len(world.players), "player(s), and",
		len(world.exits), "exit(s)")

	infoLog.Println("Starting server...")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))

	if err != nil {
		errorLog.Println("Could not start server:", err)
		return
	}

	infoLog.Println("Server listening on port", PORT)

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
