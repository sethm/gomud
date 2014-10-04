package main

import (
	"errors"
	"strings"
)

type SequentialIdGen func() int

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

//
// The world is the sum total of all objects
//
type World struct {
	idGen   SequentialIdGen
	players map[int]*Player
	rooms   map[int]*Room
	exits   map[int]*Exit
}

func NewWorld() *World {
	return &World{KeyGen(), make(map[int]*Player), make(map[int]*Room), make(map[int]*Exit)}
}

func (w *World) NewRoom(name string) (r *Room, err error) {
	normalName := strings.ToLower(name)

	r = &Room{Object: Object{key: w.idGen(), name: name, normalName: normalName},
		exits: make(map[int]*Exit), players: make(map[int]*Player)}
	w.rooms[r.key] = r
	return
}

func (w *World) NewPlayer(name string, password string, location *Room) (p *Player, err error) {
	normalName := strings.ToLower(name)

	for _, player := range w.players {
		if player.NormalName() == normalName {
			err = errors.New("User already exists")
			return
		}
	}

	p = &Player{Object: Object{key: w.idGen()}}

	p.SetName(name)
	p.SetPassword(password)
	w.players[p.key] = p
	w.MovePlayer(p, location)

	return
}

// Move a player to a new room. Returns the player's new location,
// and an error if the player could not be moved.
func (w *World) MovePlayer(p *Player, d *Room) (*Room, error) {
	p.Lock()
	defer p.Unlock()

	d.Lock()
	defer d.Unlock()

	oldRoom := p.location

	if oldRoom != nil {
		oldRoom.Lock()
		defer oldRoom.Unlock()

		delete(oldRoom.players, p.key)
	}

	p.location = d
	d.players[p.key] = p

	// Error may become non-nil in the future, when exits and rooms
	// have guards / locks

	return d, nil
}

func (w *World) NewExit(source *Room, name string, destination *Room) (e *Exit, err error) {
	normalName := strings.ToLower(name)

	for _, exit := range source.exits {
		if exit.NormalName() == normalName {
			err = errors.New("An exit with that name already exists.")
			return
		}
	}

	e = &Exit{Object: Object{key: w.idGen()}, destination: destination}
	e.SetName(name)
	w.exits[e.key] = e
	source.exits[e.key] = e

	return
}

func (world *World) connectPlayer(client *Client, player *Player) {
	client.player = player
	client.player.awake = true
	client.player.client = client
	client.Tell("Welcome, %s!", player.name)
	// world.lookHere(client)
	client.lookAt(client.player.location)
	world.TellAllButMe(client.player, player.name+" has connected.")
}

func (w *World) handleCommand(handlerMap *HandlerMap, client *Client, command Command) {
	description, exists := (*handlerMap)[command.verb]

	if !exists {
		client.Tell("Huh?")
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

	client.Tell("Huh?")
}

func (world *World) TellAllButMe(me *Player, fmt string, args ...interface{}) {
	for _, player := range me.location.players {
		client := player.client
		if client != nil && client.player != me {
			client.Tell(fmt, args...)
		}
	}
}

func (w *World) FindTarget(c *Client, cmd Command) (o Objecter, err error) {
	target := strings.ToLower(cmd.target)
	here := c.player.location

	if target == "" || target == "here" {
		o = here
		return
	}

	if target == "me" {
		o = c.player
		return
	}

	// Maybe it's an exit
	for _, e := range here.exits {
		if e.NormalName() == target {
			o = e
			return
		}
	}

	// Maybe it's a player
	for _, p := range here.players {
		if p.NormalName() == target {
			o = p
			return
		}
	}

	return nil, errors.New("Target not found")
}
