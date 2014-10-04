package main

import (
	"crypto/sha512"
	"strconv"
	"strings"
)

//
// Handlers
//

func doNewplayer(world *World, client *Client, cmd Command) {

	if cmd.target == "" || cmd.args == "" {
		client.Tell("Try: newplayer <player> <password>")
		return
	}

	normalName := strings.ToLower(cmd.target)

	for _, player := range world.players {
		if player.normalName == normalName {
			client.Tell("Sorry, that name is in use.")
			return
		}
	}

	// Ugh, what a kludge. Need a proper framework for defining
	// player creation room
	startingRoom, exists := world.rooms[1]
	if !exists {
		client.Tell("Sorry, we can't create any players right now.")
		return
	}

	player, err := world.NewPlayer(cmd.target, cmd.args, startingRoom)

	if err != nil {
		client.Tell("Sorry, we can't create any players right now.")
		return
	}

	world.connectPlayer(client, player)
}

func doConnect(world *World, client *Client, cmd Command) {
	if cmd.target == "" || cmd.args == "" {
		client.Tell("Try: connect <player> <password>")
		return
	}

	normalName := strings.ToLower(cmd.target)
	passwordHash := sha512.Sum512([]byte(cmd.args))

	for _, player := range world.players {
		if player.normalName == normalName {
			if player.password != passwordHash {
				client.Tell("Incorrect password.")
				return
			}

			// Is the player already connected?
			if player.client != nil {
				client.Tell("Already connected!")
				return
			}

			world.connectPlayer(client, player)
			return
		}
	}

	client.Tell("No such player!")
	return
}

func doSay(world *World, client *Client, cmd Command) {
	client.Tell("You say, \"" + cmd.args + "\"")
	player := client.player
	world.TellAllButMe(player, player.name+" says, \""+cmd.args+"\"")
}

func doQuit(world *World, client *Client, cmd Command) {
	client.quitRequested = true
}

func doEmote(world *World, client *Client, cmd Command) {
	player := client.player
	client.Tell(player.name + " " + cmd.args)
	world.TellAllButMe(player, player.name+" "+cmd.args)
}

func doDesc(world *World, client *Client, cmd Command) {
	desc := cmd.args

	target, err := world.FindTarget(client, cmd)

	if err != nil {
		client.Tell("I don't see that here.")
		return
	}

	if client.player != target && client.player != target.Owner() {
		client.Tell("You can't do that.")
		return
	}

	target.SetDescription(desc)
	client.Tell("Description set.")
	return
}

func doDig(world *World, client *Client, cmd Command) {
	here := client.player.location
	exitName := cmd.target
	roomName := cmd.args

	if !hasBuildPermission(client.player) {
		client.Tell("Sorry, you don't have permission to do that.")
		return
	}

	if exitName == "" || roomName == "" {
		client.Tell("Dig what?")
		return
	}

	room, err := world.NewRoom(roomName)

	if err != nil {
		client.Tell("You can't do that!")
		return
	}

	exit, _ := world.NewExit(here, exitName, room)

	room.SetOwner(client.player)
	exit.SetOwner(client.player)

	client.Tell("Dug.")
}

func doLink(world *World, client *Client, cmd Command) {
	here := client.player.location
	exitName := cmd.target

	if exitName == "" || cmd.args == "" {
		client.Tell("Dig what?")
		return
	}

	roomNumber, err := strconv.Atoi(cmd.args)
	if err != nil {
		client.Tell("I didn't understand that room number.")
		return
	}

	room, exists := world.rooms[roomNumber]

	if !exists {
		client.Tell("That destination doesn't exist.")
		return
	}

	world.NewExit(here, exitName, room)

	client.Tell("Linked.")
}

func doTell(world *World, client *Client, cmd Command) {
	client.Tell("Not Implemented Yet.")
}

func doMove(world *World, client *Client, cmd Command) {
	player := client.player
	here := player.location

	normalName := strings.ToLower(cmd.target)

	// Try to find an exit with the correct name.
	for _, exit := range here.exits {
		if exit.normalName == normalName {
			world.MovePlayer(player, exit.destination)
			client.lookAt(player.location)
			return
		}
	}

	client.Tell("There's no exit in that direction!")
}

func doHelp(world *World, client *Client, cmd Command) {
	client.Tell("Welcome to this experimental MUD!")
	client.Tell("")
	client.Tell("Basic commands are:")
	client.Tell("   go <exit>                   Move to a new room")
	client.Tell("   <direction>                 Move to a new room")
	client.Tell("   @dig <exit> <name>          Dig a new room")
	client.Tell("   @link <exit> <room_number>  Create a new exit to room #")
	client.Tell("   quit                        Leave the game")
	client.Tell("")
	client.Tell("")
}

func doExamine(world *World, client *Client, cmd Command) {
	target, err := world.FindTarget(client, cmd)

	if err != nil {
		client.Tell("I don't see that here.")
		return
	}

	client.examine(target)
}

func doSet(world *World, client *Client, cmd Command) {
	// TODO: Refactor flags into a structure with permission bits.
	if !client.player.IsSet(WizardFlag) {
		client.Tell("You don't have permission to do that!")
		return
	}

	target, err := world.FindTarget(client, cmd)

	if err != nil {
		client.Tell("I don't see that here.")
		return
	}

	if cmd.args == "" || cmd.args == "!" {
		client.Tell("What do you want to set?")
		return
	}

	isUnset := !strings.HasPrefix(cmd.args, "!")

	flagSlice := strings.SplitN(cmd.args, "!", 2)
	flagName := flagSlice[len(flagSlice)-1]

	switch flagName {
	case "builder":
		if isUnset {
			target.SetFlag(BuilderFlag)
		} else {
			target.ClearFlag(BuilderFlag)
		}
	default:
		client.Tell("I don't know that flag.")
	}
}

func doLook(world *World, client *Client, cmd Command) {
	target, err := world.FindTarget(client, cmd)

	if err != nil {
		client.Tell("I don't see that here.")
		return
	}

	client.lookAt(target)
}
