package main

import (
	"crypto/sha512"
	"strings"
	"strconv"
)

//
// Handlers
//

func doNewplayer(world *World, client *Client, cmd Command) {

	if cmd.target == "" || cmd.args == "" {
		client.tell("Try: newplayer <player> <password>")
		return
	}

	normalName := strings.ToLower(cmd.target)

	for _, player := range world.players {
		if player.normalName == normalName {
			client.tell("Sorry, that name is in use.")
			return
		}
	}

	// Ugh, what a kludge. Need a proper framework for defining
	// player creation room
	startingRoom, exists := world.rooms[1]
	if !exists {
		client.tell("Sorry, we can't create any players right now.")
		return
	}

	player, err := world.NewPlayer(cmd.target, cmd.args, startingRoom)

	if err != nil {
		client.tell("Sorry, we can't create any players right now.")
		return
	}

	world.connectPlayer(client, player)
}

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

			// Is the player already connected?
			if player.client != nil {
				client.tell("Already connected!")
				return
			}

			world.connectPlayer(client, player)
			return
		}
	}

	client.tell("No such player!")
	return

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
	client.tell(player.name + " " + cmd.args)
	world.tellAllButMe(player, player.name+" "+cmd.args)
}

func doDesc(world *World, client *Client, cmd Command) {
	desc := cmd.args

	target, err := world.FindTarget(client, cmd)

	if err != nil {
		client.tell("I don't see that here.")
		return
	}

	target.SetDescription(desc)
	client.tell("Description set.")
	return
}

func doDig(world *World, client *Client, cmd Command) {
	here := client.player.location
	exitName := cmd.target
	roomName := cmd.args

	if exitName == "" || roomName == "" {
		client.tell("Dig what?")
		return
	}

	room, err := world.NewRoom(roomName)

	if err != nil {
		client.tell("You can't do that!")
		return
	}

	world.NewExit(here, exitName, room)

	client.tell("Dug.")
}

func doLink(world *World, client *Client, cmd Command) {
	here := client.player.location
	exitName := cmd.target

	if exitName == "" || cmd.args == "" {
		client.tell("Dig what?")
		return
	}

	roomNumber, err := strconv.Atoi(cmd.args)
	if err != nil {
		client.tell("I didn't understand that room number.")
		return
	}

	room, exists := world.rooms[roomNumber]

	if !exists {
		client.tell("That destination doesn't exist.")
		return
	}

	world.NewExit(here, exitName, room)

	client.tell("Linked.")
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

func doHelp(world *World, client *Client, cmd Command) {
	client.tell("Welcome to this experimental MUD!")
	client.tell("")
	client.tell("Basic commands are:")
	client.tell("   go <exit>                   Move to a new room")
	client.tell("   <direction>                 Move to a new room")
	client.tell("   @dig <exit> <name>          Dig a new room")
	client.tell("   @link <exit> <room_number>  Create a new exit to room #")
	client.tell("   quit                        Leave the game")
	client.tell("")
	client.tell("")

}

func doLook(world *World, client *Client, cmd Command) {
	if cmd.target == "" || cmd.target == "here" {
		world.lookHere(client)
		return
	}

	if cmd.target == "me" {
		client.tell(client.player.Description())
		return
	}

	player := client.player
	here := player.location

	// Maybe it's a player?
	for _, p := range here.players {
		if cmd.target == p.name {
			client.tell(p.Description())
			return
		}
	}

	// Not a player, maybe an exit
	for _, e := range here.exits {
		if cmd.target == e.name {
			client.tell(e.Description())
			return
		}
	}

	client.tell("I don't see that here.")
}
