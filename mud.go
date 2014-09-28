package main

import (
	"crypto/sha512"
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

var world *World = NewWorld()
var debugLog, infoLog, errorLog *log.Logger
var idGen func() int = KeyGen()

type CommandHandler func(*World, *Client, Command)

type CommandDesc struct {
	argCount int
	preAuth  bool
	postAuth bool
	handler  CommandHandler
}

type HandlerMap map[string]CommandDesc

var commandHandlers = HandlerMap{
	"@desc":   {2, false, true, doDesc},
	"connect": {2, true, false, doConnect},
	"emote":   {1, false, true, doEmote},
	"go":      {1, false, true, doMove},
	"look":    {1, false, true, doLook},
	"move":    {1, false, true, doMove},
	"quit":    {0, true, true, doQuit},
	"say":     {1, false, true, doSay},
	"tell":    {2, false, true, doTell},
	"walk":    {1, false, true, doMove},
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
	conn          net.Conn
	player        *Player
	quitRequested bool
}

//
// Exits link two rooms together
//
type Exit struct {
	key         int
	name        string
	normalName  string
	description string
	owner       *Player
	destination *Room
}

//
// A room is a place in the world.
//
type Room struct {
	key         int
	name        string
	description string
	owner       *Player
	exits       map[int]*Exit
	players     map[int]*Player
}

//
// A player interacts with the world
//
type Player struct {
	key         int
	name        string
	password    [64]byte
	description string
	normalName  string
	location    *Room
	awake       bool
	client      *Client
}

func (p *Player) Description() string {
	if p.description == "" {
		return "You see nothing special."
	}
	
	return p.description
}

func (p *Player) SetPassword(raw string) {
	p.password = sha512.Sum512([]byte(raw))
}

//
// The world is the sum total of all objects
//
type World struct {
	players map[int]*Player
	rooms   map[int]*Room
	exits   map[int]*Exit
}

func NewWorld() *World {
	return &World{make(map[int]*Player), make(map[int]*Room), make(map[int]*Exit)}
}

func (w *World) NewRoom(name string) (r *Room, err error) {
	r = &Room{key: idGen(), name: name, exits: make(map[int]*Exit), players: make(map[int]*Player)}
	w.rooms[r.key] = r
	return
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn, quitRequested: false}
}

func (c *Client) tell(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg+"\r\n", args...)
	c.conn.Write([]byte(s))
}

func (w *World) NewPlayer(name string, password string, location *Room) (p *Player, err error) {
	normalName := strings.ToLower(name)

	for _, player := range w.players {
		if player.normalName == normalName {
			err = errors.New("User already exists")
			return
		}
	}

	p = &Player{key: idGen(), name: name, normalName: normalName}
	p.SetPassword(password)
	w.players[p.key] = p
	w.MovePlayer(p, location)

	return
}

// Move a player to a new room. Returns the player's new location,
// and an error if the player could not be moved.
func (w *World) MovePlayer(p *Player, destination *Room) (r *Room, err error) {
	oldRoom := p.location
	if oldRoom != nil {
		delete(oldRoom.players, p.key)
	}
	p.location = destination
	destination.players[p.key] = p
	return destination, nil
}

func (w *World) NewExit(source *Room, name string, destination *Room) (e *Exit, err error) {
	for _, exit := range source.exits {
		if exit.name == name {
			err = errors.New("An exit with that name already exists.")
			return
		}
	}

	e = &Exit{key: idGen(), name: name, destination: destination}
	w.exits[e.key] = e
	source.exits[e.key] = e

	return
}

// TODO: I feel like this needs improvement.
func (w *World) parseCommand(client *Client, line string) Command {
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

func (w *World) handleCommand(handlerMap *HandlerMap, client *Client, command Command) {

	description, exists := (*handlerMap)[command.verb]

	if !exists {
		client.tell("Huh?")
		return
	}

	// Are we pre-auth?
	if client.player == nil && description.preAuth {
		// OK to handle.
		description.handler(w, client, command)
		return
	}

	// ... or are we post-auth?
	if client.player != nil && description.postAuth {
		description.handler(w, client, command)
		return
	}

	client.tell("Huh?")
}

//
// Handlers
//

func doConnect(world *World, client *Client, cmd Command) {

	if cmd.target == "" || cmd.args == "" {
		client.tell("Try: connect <player> <password>")
		return
	}
	
	normalName := strings.ToLower(cmd.target)
	passwordHash := sha512.Sum512([]byte(cmd.args))

	for _, player := range world.players {
		if player.normalName == normalName {

			if player.password != passwordHash {
				client.tell("Incorrect password.")
				return
			}
			
			client.player = player
			client.player.awake = true
			client.player.client = client
			client.tell("Welcome, %s!", player.name)
			world.lookHere(client)

			world.tellAllButMe(client.player, player.name+" has connected.")
			return
		}
	}

	client.tell("No such player!")
	return

}

func (world *World) tellAllButMe(me *Player, fmt string, args ...interface{}) {
	for _, player := range me.location.players {
		client := player.client
		if client != nil && client.player != me {
			client.tell(fmt, args...)
		}
	}
}

func doSay(world *World, client *Client, cmd Command) {
	client.tell("You say, \"" + cmd.args + "\"")
	player := client.player
	world.tellAllButMe(player, player.name+" says, \""+cmd.args+"\"")
}

func doQuit(world *World, client *Client, cmd Command) {
	client.quitRequested = true
}

func doEmote(world *World, client *Client, cmd Command) {
	player := client.player
	client.tell(player.name+" "+cmd.args)
	world.tellAllButMe(player, player.name+" "+cmd.args)
}

func doDesc(world *World, client *Client, cmd Command) {
	player := client.player
	here := player.location
	here.description = cmd.args
	client.tell("Set.")
}

func doTell(world *World, client *Client, cmd Command) {
	client.tell("Not Implemented Yet.")
}

func doMove(world *World, client *Client, cmd Command) {
	player := client.player
	here := player.location

	// Try to find an exit with the correct name.
	for _, exit := range here.exits {
		if exit.name == cmd.args {
			world.MovePlayer(player, exit.destination)
			world.lookHere(client)
			return
		}
	}

	client.tell("There's no exit in that direction!")
}

func (world *World) lookHere(client *Client) {
	player := client.player
	here := player.location
	client.tell("%s", here.name)

	if here.description != "" {
		client.tell("\n" + here.description + "\n")
	}

	if len(here.exits) > 0 {
		client.tell("You can see the following exits:")
		for _, exit := range here.exits {
			client.tell("  %s", exit.name)
		}
	}

	if len(here.players) > 0 {
		client.tell("The following players are here:")
		for _, p := range here.players {
			if p.normalName != player.normalName {
				if p.awake {
					client.tell("  %s", p.name)
				} else {
					client.tell("  %s (asleep)", p.name)
				}
			}
		}
	}
}

func doLook(world *World, client *Client, cmd Command) {
	if cmd.args == "" || cmd.args == "here" {
		world.lookHere(client)
		return
	}

	if cmd.args == "me" {
		client.tell(client.player.Description())
		return
	}

	client.tell("I don't see that here.")

}

func welcome(client *Client) {
	client.tell("-----------------------------------------------------")
	client.tell("Welcome to this experimental MUD!")
	client.tell("")
	client.tell("To create a new player: create <name> <password>")
	client.tell("To connect as a player: connect <name> <password>")
	client.tell("To leave the game:      quit")
	client.tell("-----------------------------------------------------")
	client.tell("")
	client.tell("")
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
		// client.tell("mud> ")
		n, err := conn.Read(linebuf)

		if err != nil {
			if err != io.EOF {
				errorLog.Println("Error:", err)
			}
			break
		}

		line := strings.TrimSpace(string(linebuf[:n]))

		if len(line) > 0 {
			command := world.parseCommand(client, line)
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
		world.tellAllButMe(client.player, client.player.name+" has disconnected.")
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
		stopRequested <- true
	}()

	infoLog.Println("Starting server...")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))

	if err != nil {
		errorLog.Println("Could not start server:", err)
		return
	}

	infoLog.Println("Server listening on port", PORT)

	initWorld()

	infoLog.Println("World initialized with",
		len(world.rooms), "room(s),",
		len(world.players), "player(s), and",
		len(world.exits), "exit(s)")

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
