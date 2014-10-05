package main

import (
	"testing"
)

func TestDoConnectShouldWakeUpPlayers(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	if bob.awake {
		t.Errorf("Bob should not be awake.")
	}

	doConnect(world, client, Command{"connect", "", "bob foo"})

	if client.player != bob {
		t.Errorf("Connecting should have linked the client and the player")
	}

	if !bob.awake {
		t.Errorf("Connecting should have woken up bob.")
	}
}

func TestDoConnectShouldNotConnectMultipleTimes(t *testing.T) {
	world := NewWorld()

	connA := NewMockConn()
	clientA := NewClient(connA)

	connB := NewMockConn()
	clientB := NewClient(connB)

	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, clientA, Command{"connect", "", "bob foo"})

	if clientA.player != bob || bob.client != clientA {
		t.Errorf("Connecting should have linked the client and the player")
	}

	doConnect(world, clientB, Command{"connect", "", "bob foo"})

	if clientB.player == bob || bob.client == clientB {
		t.Errorf("Should NOT be able to connect to the same player twice.")
	}
}

func TestDoConnectDoesNothingIfPlayerNotFound(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn)
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "", "jim foo"})

	assertMatch(t, "No such player!\r\n", conn.String())

	if bob.awake {
		t.Errorf("Bob should still be asleep.")
	}
}

func TestDoConnectWithoutPasswordFails(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn)
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "", "bob"})

	assertMatch(t, "Try: connect <player> <password>\r\n", conn.String())

	if bob.awake {
		t.Errorf("Bob should still be asleep.")
	}
}

func TestDoConnectWithWrongPasswordFails(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn)
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "", "bob bar"})

	assertMatch(t, "Incorrect password.\r\n", conn.String())

	if bob.awake {
		t.Errorf("Bob should still be asleep.")
	}
}

func TestDoConnectWithoutUserFails(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "", ""})

	assertMatch(t, "Try: connect <player> <password>\r\n", conn.String())

	if bob.awake {
		t.Errorf("Bob should still be asleep.")
	}
}

func TestDoSay(t *testing.T) {
	world := NewWorld()

	bobConn := NewMockConn()
	jimConn := NewMockConn()
	bobClient := NewClient(bobConn)
	jimClient := NewClient(jimConn)

	hall, _ := world.NewRoom("The Hall")

	world.NewPlayer("bob", "foo", hall)
	world.NewPlayer("jim", "foo", hall)

	doConnect(world, bobClient, Command{"connect", "", "bob foo"})
	doConnect(world, jimClient, Command{"connect", "", "jim foo"})

	doSay(world, bobClient, Command{"say", "", "Testing 1 2 3"})

	assertMatch(t, "You say, \"Testing 1 2 3\"\r\n", bobConn.String())
	assertMatch(t, "bob says, \"Testing 1 2 3\"\r\n", jimConn.String())
}

func TestDoEmote(t *testing.T) {
	world := NewWorld()

	bobConn := NewMockConn()
	jimConn := NewMockConn()
	bobClient := NewClient(bobConn)
	jimClient := NewClient(jimConn)

	hall, _ := world.NewRoom("The Hall")

	world.NewPlayer("bob", "foo", hall)
	world.NewPlayer("jim", "foo", hall)

	doConnect(world, bobClient, Command{"connect", "", "bob foo"})
	doConnect(world, jimClient, Command{"connect", "", "jim foo"})

	doEmote(world, bobClient, Command{"emote", "", "tests."})

	assertMatch(t, "bob tests.\r\n", bobConn.String())
	assertMatch(t, "bob tests.\r\n", jimConn.String())
}

func TestDoQuit(t *testing.T) {
	world := NewWorld()

	conn := NewMockConn()
	client := NewClient(conn)

	if client.quitRequested {
		t.Errorf("Expected 'quitRequested' to have been false.")
	}

	doQuit(world, client, Command{"quit", "", ""})

	if !client.quitRequested {
		t.Errorf("Expected 'quitRequested' to have been set.")
	}
}

func TestDoLookShowsHereByDefault(t *testing.T) {
	world := NewWorld()

	bobConn := NewMockConn()
	sallyConn := NewMockConn()
	bobClient := NewClient(bobConn)
	sallyClient := NewClient(sallyConn)

	hall, _ := world.NewRoom("The Hall")
	den, _ := world.NewRoom("The Den")

	hall.description = "It's a lovely hall"

	world.NewExit(hall, "east", den)
	world.NewExit(den, "west", hall)

	world.NewRoom("The Den")

	world.NewPlayer("bob", "foo", hall)
	world.NewPlayer("jim", "foo", hall)
	world.NewPlayer("sally", "foo", hall)

	doConnect(world, bobClient, Command{"connect", "", "bob foo"})
	doConnect(world, sallyClient, Command{"connect", "", "sally foo"})

	doLook(world, bobClient, Command{"look", "", ""})

	// Should see the name of the room
	assertMatch(t, "The Hall", bobConn.String())
	// Should see the description of the room
	assertMatch(t, "It's a lovely hall", bobConn.String())
	// Should see the exit
	assertMatch(t, "east", bobConn.String())
	// Should see jim (asleep)
	assertMatch(t, "jim \\(asleep\\)\r\n", bobConn.String())
	// Should see sally (awake)
	assertMatch(t, "sally\r\n", bobConn.String())
}

func TestDoDigCreatesRoom(t *testing.T) {
	world := NewWorld()
	conn := NewMockConn()
	client := NewClient(conn)
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)
	hall.SetOwner(bob)
	bob.SetFlag(BuilderFlag)

	doConnect(world, client, Command{"connect", "", "bob foo"})

	if len(world.rooms) != 1 {
		t.Errorf("There should be only 1 room")
	}

	if len(world.exits) != 0 {
		t.Errorf("There should be no exits.")
	}

	doDig(world, client, Command{"@dig", "east", "The Den"})

	if len(world.rooms) != 2 {
		t.Errorf("Den was not created.")
	}

	if len(world.exits) != 1 {
		t.Errorf("Exit was not created.")
	}
}

func TestDoDigSetsOwnershipOfNewRoom(t *testing.T) {
	world := NewWorld()
	conn := NewMockConn()
	client := NewClient(conn)
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)
	hall.SetOwner(bob)
	bob.SetFlag(BuilderFlag)
	doConnect(world, client, Command{"connect", "", "bob foo"})
	doDig(world, client, Command{"@dig", "east", "The Den"})

	// The new room should be db #3
	den, exists := world.rooms[3]

	if !exists || den.name != "The Den" {
		t.Errorf("Expected to find The Den")
	}

	if den.owner != bob {
		t.Errorf("Expected bob to own The Den")
	}

	exit, exists := world.exits[4]

	if !exists {
		t.Errorf("Expected to find an exit.")
	}

	if exit.owner != bob {
		t.Errorf("Expected bob to own the exit")
	}
}

func TestDoDescriptionUpdatesDescription(t *testing.T) {
	world := NewWorld()
	conn := NewMockConn()
	client := NewClient(conn)
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)
	hall.SetOwner(bob)

	doConnect(world, client, Command{"connect", "", "bob foo"})

	doDesc(world, client, Command{"@desc", "me", "Bob is really tall."})

	if bob.description != "Bob is really tall." {
		t.Errorf("Bob's description was not updated: " + bob.Description())
	}
}

func TestDoDescriptionOnlyUpdatesDescIfPlayerIsTheOwner(t *testing.T) {
	world := NewWorld()
	bobConn := NewMockConn()
	bobClient := NewClient(bobConn)
	jimConn := NewMockConn()
	jimClient := NewClient(jimConn)
	hall, _ := world.NewRoom("The Hall")
	hall.SetDescription("The Hall Is Dark")
	world.NewPlayer("bob", "foo", hall)
	jim, _ := world.NewPlayer("jim", "foo", hall)
	hall.SetOwner(jim)

	doConnect(world, bobClient, Command{"connect", "", "bob foo"})
	doConnect(world, jimClient, Command{"connect", "", "jim foo"})
	doDesc(world, bobClient, Command{"@desc", "here", "The Hallway is long"})

	if hall.Description() != "The Hall Is Dark" {
		t.Errorf("Bob should not be able to change the desc of the room.")
	}

	doDesc(world, jimClient, Command{"@desc", "here", "The Hallway is short"})

	if hall.Description() != "The Hallway is short" {
		t.Errorf("Jim should be able to change the desc of the room.")
	}
}

func TestDoSetSetsBuilderFlag(t *testing.T) {
	world := NewWorld()
	wizardConn := NewMockConn()
	wizardClient := NewClient(wizardConn)
	hall, _ := world.NewRoom("The Hall")
	hall.SetDescription("The Hall Is Dark")
	wizard, _ := world.NewPlayer("wizard", "foo", hall)
	jim, _ := world.NewPlayer("jim", "foo", hall)

	wizard.SetFlag(WizardFlag)

	doConnect(world, wizardClient, Command{"connect", "", "wizard foo"})

	if jim.IsSet(BuilderFlag) {
		t.Errorf("Jim should not be a builder.")
	}

	doSet(world, wizardClient, Command{"@set", "jim", "builder"})

	if !jim.IsSet(BuilderFlag) {
		t.Errorf("Jim should be a builder.")
	}

	doSet(world, wizardClient, Command{"@set", "jim", "!builder"})

	if jim.IsSet(BuilderFlag) {
		t.Errorf("Jim should not be a builder.")
	}
}

func TestDoSetBuilderFlagRequiresWizardPermissions(t *testing.T) {
	world := NewWorld()
	jimConn := NewMockConn()
	jimClient := NewClient(jimConn)
	hall, _ := world.NewRoom("The Hall")
	hall.SetDescription("The Hall Is Dark")
	wizard, _ := world.NewPlayer("wizard", "foo", hall)
	world.NewPlayer("jim", "foo", hall)

	wizard.SetFlag(WizardFlag)
	wizard.SetFlag(BuilderFlag)

	doConnect(world, jimClient, Command{"connect", "", "jim foo"})

	if !wizard.IsSet(BuilderFlag) {
		t.Errorf("Wizard should be a builder at start.")
	}

	// Jim isn't a wizard, he can't do this.
	doSet(world, jimClient, Command{"@set", "wizard", "!builder"})

	if !wizard.IsSet(BuilderFlag) {
		t.Errorf("Wizard should still be a builder after trying to unset flag")
	}

}
