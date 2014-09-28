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

var world *World = NewWorld()
var connections Set = NewSet()
var debugLog, infoLog, errorLog *log.Logger
var idGen func() int = KeyGen()

type CommandHandler func(*World, *Client, Command)

type HandlerMap map[string]CommandHandler

var argCounts = map[string]int{
	"connect": 1,
	"quit":    0,
	"go":      1,
	"walk":    1,
	"move":    1,
	"say":     1,
	"emote":   1,
	"@desc":   2,
}

var preAuthHandlers = HandlerMap{
	"connect": doConnect,
	"quit":    doQuit,
}

var postAuthHandlers = HandlerMap{
	"go":    doMove,
	"walk":  doMove,
	"move":  doMove,
	"say":   doSay,
	"emote": doEmote,
	"look":  doLook,
	"@desc": doDesc,
	"quit":  doQuit,
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

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn, quitRequested: false}
}

func (c *Client) tell(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg+"\r\n", args...)
	c.conn.Write([]byte(s))
}

//
// Exits link two rooms together
//
type Exit struct {
	key               int
	name, description string
	destination       *Room
}

// Exit implements Object interface
func (e Exit) Key() int            { return e.key }
func (e Exit) Name() string        { return e.name }
func (e Exit) Description() string { return e.description }

//
// A room is a place in the world.
//
type Room struct {
	key               int
	name, description string
	exits             Set
}

// Room implements Object interface
func (r Room) Key() int            { return r.key }
func (r Room) Name() string        { return r.name }
func (r Room) Description() string { return r.description }

type Player struct {
	key               int
	name, description string
	location          *Room
	awake             bool
	client            *Client
}

// Player implements Object interface
func (p Player) Key() int            { return p.key }
func (p Player) Name() string        { return p.name }
func (p Player) Description() string { return p.description }
func (p *Player) Awake() bool        { return p.awake }

type World struct {
	players Set
	rooms   Set
	exits   Set
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
	if w.players.ContainsWhere(func(o Object) bool { return o.Name() == name }) {
		err = errors.New("User already exists")
	} else {
		p = &Player{key: idGen(), name: name, location: location}
		w.players.Add(p)
	}

	return
}

func (w *World) NewExit(source *Room, name string, destination *Room) (e *Exit, err error) {
	foundExit := source.exits.ContainsWhere(func(o Object) bool {
		return o.Name() == name
	})

	if foundExit {
		err = errors.New("An exit with that name already exists.")
	} else {
		e = &Exit{key: idGen(), name: name, destination: destination}
		w.exits.Add(e)
		source.exits.Add(e)
	}

	return
}

// TODO: I feel like this needs improvement.
func (w *World) parseCommand(client *Client, line string) Command {
	// The user may have typed `"foo`, which we want to interpret
	// as "say foo".
	if strings.HasPrefix(line, "\"") {
		return Command{verb: "say", args: line[1:len(line)]}
	} else if strings.HasPrefix(line, ":") {
		return Command{verb: "emote", args: line[1:len(line)]}
	} else {
		tokenized := strings.SplitN(line, " ", 2)
		if len(tokenized) == 2 {
			return Command{verb: tokenized[0], args: tokenized[1]}
		} else {
			foundExit := false

			// If the player is connected, do some special magic.
			if client.player != nil {
				// The user may have typed an exit name as a command. In that
				// case, we want to interpret what she's said as a `move`
				// command
				location := client.player.location

				for exit := range location.exits.Iterator() {
					if tokenized[0] == exit.Name() {
						foundExit = true
					}
				}
			}

			if foundExit {
				return Command{verb: "move", args: tokenized[0]}
			} else {
				return Command{verb: tokenized[0]}
			}
		}
	}
}

func (w *World) handleCommand(preAuthHandlers *HandlerMap, postAuthHandlers *HandlerMap, client *Client, command Command) {

	if client.player == nil {
		w.dispatchToHandler((*preAuthHandlers)[command.verb], client, command)
		return
	}

	w.dispatchToHandler((*postAuthHandlers)[command.verb], client, command)
}

func (w *World) dispatchToHandler(handler CommandHandler, client *Client, cmd Command) {
	if handler == nil {
		client.tell("Huh?")
	} else {
		handler(w, client, cmd)
	}
}

//
// Handlers
//

// TODO: Refactor and clean up the 'if client.player' stuff

func doConnect(world *World, client *Client, cmd Command) {
	if client.player != nil {
		return
	}

	player := world.players.SelectFirst(func(o Object) bool {
		return o.Name() == cmd.args
	})

	if player == nil {
		client.tell("No such player!")
		return
	}

	// We must use a type assertion to cast Object to type *Player
	client.player = player.(*Player)
	client.player.awake = true
	client.player.client = client
	client.tell("Welcome, %s!", player.Name())

	world.tellAllButMe(client.player, player.Name() + " has connected.")

}

func (world *World) tellAllButMe(me *Player, fmt string, args ...interface{}) {
	players := world.PlayersAt(me.location)

	for _, player := range players {
		client := player.client
		if client != nil && client.player != me {
			client.tell(fmt, args...)
		}
	}
}

func doSay(world *World, client *Client, cmd Command) {
	client.tell("You say, \"" + cmd.args + "\"")
	player := client.player
	world.tellAllButMe(player, player.Name() + " says, \"" + cmd.args + "\"")
}

func doQuit(world *World, client *Client, cmd Command) {
	client.quitRequested = true
}

func doEmote(world *World, client *Client, cmd Command) {
	client.tell(client.player.name + " " + cmd.args)
}

func doDesc(world *World, client *Client, cmd Command) {
	player := client.player
	here := player.location
	here.description = cmd.args
	client.tell("Set.")
}

func doMove(world *World, client *Client, cmd Command) {
	player := client.player
	here := player.location

	// Try to find an exit with the correct name.
	for exit := range here.exits.Iterator() {
		if exit.Name() == cmd.args {
			player.location = exit.(*Exit).destination
			lookHere(world, client)
			return
		}
	}

	client.tell("There's no exit in that direction!")
}

func (w *World) PlayersAt(room *Room) []*Player {
	// Must use type assertions because the collection is
	// of type Object, not type *Player. Ugh.
	objects := world.players.Select(func(o Object) bool {
		return o.(*Player).location == room
	})

	players := make([]*Player, len(objects), len(objects))
	for i, o := range objects {
		players[i] = o.(*Player)
	}
	return players
}

func lookHere(world *World, client *Client) {
	player := client.player
	here := player.location
	client.tell("You are in: %s", here.name)

	if here.description != "" {
		client.tell("\n" + here.description + "\n")
	}

	if here.exits.Len() > 0 {
		client.tell("You can see the following exits:")
		for exit := range here.exits.Iterator() {
			client.tell("  %s", exit.Name())
		}
	}

	// TODO: Do we want to denormalize this? i.e., we'd have
	// a circular relationship where a room has a collection of
	// player pointers, and a player has a room pointer?
	// Ripe for a refactor.
	players := world.PlayersAt(here)

	if len(players) > 0 {
		client.tell("The following players are here:")
		for _, p := range players {
			if p.Name() != player.Name() {
				if p.Awake() {
					client.tell("  %s", p.Name())
				} else {
					client.tell("  %s (asleep)", p.Name())
				}
			}
		}
	}

}

func doLook(world *World, client *Client, cmd Command) {
	if cmd.args == "" {
		lookHere(world, client)
	} else {
		// TODO: Refactor when there are objects
		client.tell("I don't see that here.")
	}
}

func welcome(client *Client) {
	client.tell("-----------------------------------------------------")
	client.tell("Welcome to this experimental MUD!")
	client.tell("")
	client.tell("To create a new player: create <player_name>")
	client.tell("To connect as a player: connect <player_name>")
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
		debugLog.Println(fmt.Sprintf("[%s]: %s", conn.RemoteAddr(), line))

		command := world.parseCommand(client, line)
		world.handleCommand(&preAuthHandlers, &postAuthHandlers, client, command)

		if client.quitRequested {
			break
		}
	}

	infoLog.Println("Disconnection from", conn.RemoteAddr())

	world.tellAllButMe(client.player, client.player.Name() + " has disconnected.")

	client.player.awake = false
	client.player.client = nil
	client.player = nil

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

	world.NewPlayer("god", hall)
	world.NewPlayer("wizard", hall)
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

	infoLog.Println("World initialized with", world.rooms.Len(), "room(s) and", world.players.Len(), "player(s)")

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
