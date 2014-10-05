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

type CommandHandler func(*World, *Client, Command)

type CmdType uint8

const (
	UnaryCmd CmdType = iota
	ArgsCmd
	TargetedCmd
)

type CommandDesc struct {
	cmdType  CmdType
	preAuth  bool
	postAuth bool
	handler  CommandHandler
}

type HandlerMap map[string]CommandDesc

var commandHandlers = HandlerMap{
	"@desc":     {TargetedCmd, false, true, doDesc},
	"@dig":      {TargetedCmd, false, true, doDig},
	"@help":     {UnaryCmd, false, true, doHelp},
	"@link":     {TargetedCmd, false, true, doLink},
	"connect":   {ArgsCmd, true, false, doConnect},
	"examine":   {TargetedCmd, false, true, doExamine},
	"ex":        {TargetedCmd, false, true, doExamine},
	"newplayer": {TargetedCmd, true, false, doNewplayer},
	"emote":     {ArgsCmd, false, true, doEmote},
	"go":        {TargetedCmd, false, true, doMove},
	"help":      {UnaryCmd, false, true, doHelp},
	"look":      {TargetedCmd, false, true, doLook},
	"l":         {TargetedCmd, false, true, doLook},
	"move":      {TargetedCmd, false, true, doMove},
	"quit":      {UnaryCmd, true, true, doQuit},
	"say":       {ArgsCmd, false, true, doSay},
	"@set":      {TargetedCmd, false, true, doSet},
	"tell":      {TargetedCmd, false, true, doTell},
	"walk":      {TargetedCmd, false, true, doMove},
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

func (client *Client) examine(o Objecter) {
	client.Tell("%s (#%d)", o.Name(), o.Key())

	if o.Owner() != nil {
		client.Tell("Owner: %s (#%d)", o.Owner().Name(), o.Owner().Key())
	}
}

func (client *Client) lookAt(o Objecter) {
	player := client.player

	client.Tell("%s (#%d)", o.Name(), o.Key())
	client.Tell(o.Description())

	// If the Object is a room, we want more info.
	switch o.(type) {
	case *Room:
		r := o.(*Room)

		if len(r.exits) > 0 {
			client.Tell("You can see the following exits:")
			for _, exit := range r.exits {
				client.Tell("  %s", exit.name)
			}
		}

		if len(r.players) > 1 {
			client.Tell("The following players are here:")
			for _, p := range r.players {
				if p.NormalName() != player.NormalName() {
					if p.awake {
						client.Tell("  %s", p.name)
					} else {
						client.Tell("  %s (asleep)", p.name)
					}
				}
			}
		}
	}
}

func parseCommand(client *Client, line string) Command {

	// First up, we do some special processing to normalize the input,
	// for the case where the user may be typing a command like '"foo'
	// as shortcut for 'say foo', or ':bar' as a shortcut for 'emote
	// bar'. This is a hack, but a useful one.

	if strings.HasPrefix(line, "\"") {
		line = "say " + line[1:len(line)]
	} else if strings.HasPrefix(line, ":") {
		line = "emote " + line[1:len(line)]
	}

	// Now we further tokenize the line into VERB and ARGS

	tokenized := strings.SplitN(line, " ", 2)

	verb := tokenized[0]

	info, isKeyword := commandHandlers[verb]

	// Now we have a further complication. We allow the user to use a
	// shortcut for moving around the world. For example, if there is
	// an exit named "west", the user can just type "west" to move
	// there. If the verb is the name of a direction, we short-circuit
	// and return a Command of the right form. We only do this,
	// though, if the command is not a keyword.

	if !isKeyword && client.player != nil {
		location := client.player.location
		for _, exit := range location.exits {
			if tokenized[0] == exit.name {
				return Command{verb: "move", target: tokenized[0]}
			}
		}
	}

	// Now with that out of the way, we can proceed to tokenize the
	// rest of the command appropriately.

	// No idea what to do, it's not a keyord. Return 0-command.
	if !isKeyword {
		return Command{}
	}

	if len(tokenized) == 1 {
		return Command{verb: verb}
	}

	if info.cmdType == ArgsCmd {
		return Command{verb: verb, args: tokenized[1]}
	}

	if info.cmdType == TargetedCmd {
		// Further tokenize the args into target/args
		argTokens := strings.SplitN(tokenized[1], "=", 2)

		if len(argTokens) == 1 {
			return Command{verb: verb, target: argTokens[0]}
		}

		return Command{verb: verb, target: argTokens[0], args: argTokens[1]}
	}

	return Command{} // Catch-all, 0-command
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
	helm, _ := world.NewRoom("Wizard's Helm")

	wizard, _ := world.NewPlayer("Wizard", "xyzzy", helm)
	wizard.SetFlag(WizardFlag)
	wizard.SetFlag(BuilderFlag)

	helm.owner = wizard
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
