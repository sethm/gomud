package main

import (
	"bytes"
	"net"
	"testing"
	"time"
	"regexp"
)

func assertMatch(t *testing.T, expected string, match string) {
	exp, _ := regexp.Compile(expected)
	if !exp.MatchString(match) {
		t.Errorf("String match not found. Expected: '%s', Actual: '%s'", expected, match)
	}
}

//
// Implement a mock version of net.Conn for testing
//

type MockConn struct {
	readBytes    [][]byte
	writeBuffer  *bytes.Buffer

	readError        *error
	closeAfterWrites int
	numWrites        int
}

func (conn MockConn) Read(b []byte) (n int, err error) {
	read := copy(b, conn.readBytes[conn.numWrites])
	conn.numWrites++
	return read, nil
}

func (conn MockConn) Write(b []byte) (n int, err error) {
	// conn.writeBuffer.Reset()
	n, err = conn.writeBuffer.Write(b)
	return
}

func (conn MockConn) Close() error {
	return nil
}

func (conn MockConn) LocalAddr() net.Addr {
	return &net.IPAddr{net.IPv4(192, 168, 1, 1), ""}
}

func (conn MockConn) RemoteAddr() net.Addr {
	return &net.IPAddr{net.IPv4(192, 168, 1, 1), ""}
}

func (conn MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (conn MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func NewMockConn() *MockConn {
	buffer := bytes.NewBuffer(make([]byte, 1024, 1024))
	return &MockConn{readBytes: make([][]byte, 1024, 1024), writeBuffer: buffer}
}


// Automatically convert writtenBytes into a string
func (conn MockConn) String() string {
	return conn.writeBuffer.String()
}

func TestKeyGen(t *testing.T) {
	gen := KeyGen()

	i := gen()
	j := gen()
	k := gen()

	if i != 1 || j != 2 || k != 3 {
		t.Errorf("Incorrect sequence generated by KeyGen")
	}

}

func TestNewWorld(t *testing.T) {
	w := NewWorld()

	if len(w.players) != 0 {
		t.Errorf("Expected world to exist and have 0 players.")
	}
}

func TestNewRoom(t *testing.T) {
	world := NewWorld()
	hall, err := world.NewRoom("The Hall")

	if hall.name != "The Hall" {
		t.Errorf("Expected name to be 'The Hall'")
	}
	if hall == nil || err != nil {
		t.Errorf("Expected to create a new room.")
	}
	if _, hasHall := world.rooms[hall.key]; len(world.rooms) != 1 || !hasHall {
		t.Errorf("Expected room to have been added to the world.")
	}
}

func TestNewPlayer(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)
	if bob.name != "bob" {
		t.Errorf("Expected player name to be bob, but was %s", bob.name)
	}
	if bob.location != hall {
		t.Errorf("Expected player's location to be the hall")
	}
	if _, containsBob := world.players[bob.key]; len(world.players) != 1 || !containsBob {
		t.Errorf("Expected player to have been added to world.")
	}

	// Should store a normalized name
	jim, _ := world.NewPlayer("JiM", "foo", hall)
	if jim.name != "JiM" {
		t.Errorf("Expected jim's display name to be 'JiM'")
	}
	if jim.normalName != "jim" {
		t.Errorf("Expected jim's normal name to be 'jim'")
	}
}

func TestNewPlayerCantReuseNames(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	world.NewPlayer("bob", "foo", hall)
	otherBob, err := world.NewPlayer("bob", "foo", hall)
	if otherBob != nil || err == nil {
		t.Errorf("Should not have been able to create duplicate user.")
	}
}

func TestTell(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	client.tell("Hello, world!")

	assertMatch(t, "Hello, world!\r\n", conn.String())
}

func TestNewExit(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	den, _ := world.NewRoom("The Den")

	east, err1 := world.NewExit(hall, "east", den)
	west, err2 := world.NewExit(den, "west", hall)

	if east.name != "east" {
		t.Errorf("Expected hall exit to be named 'east'")
	}
	if west.name != "west" {
		t.Errorf("Expected hall exit to be named 'east'")
	}
	if err1 != nil || err2 != nil {
		t.Errorf("Error while creating exits.")
	}
}

func TestNewExitAddsToWorldSet(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	den, _ := world.NewRoom("The Den")

	world.NewExit(hall, "east", den)
	world.NewExit(den, "west", hall)

	if len(world.exits) != 2 {
		t.Errorf("Exits were not added to the global set:")
	}
}

func TestNewExitFailsWhenCreatingDuplicateExits(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	den, _ := world.NewRoom("The Den")

	world.NewExit(hall, "east", den)
	exit, err := world.NewExit(hall, "east", hall)

	if exit != nil || err == nil {
		t.Errorf("Creating exit should have failed.")
	}
}

var commandInputs = []string{
	"",
	"look",
	"walk east",
	"west", // There's an exit to the west
	"east", // No such exit
	"say",  // There's an exit named 'say', but that's a keyword!
	"say foo bar baz",
	"@desc me I'm very tall",
	"tell bob Hey there!",
	"\"hey bob",
	":waves hello.",
	"",
	"tell bob", // Valid, but missing arg
}

var expectedCommands = []Command{
	{"", "", ""},
	{"look", "", ""},
	{"walk", "", "east"},
	{"move", "", "west"},
	{"east", "", ""},
	{"say", "", ""},
	{"say", "", "foo bar baz"},
	{"@desc", "me", "I'm very tall"},
	{"tell", "bob", "Hey there!"},
	{"say", "", "hey bob"},
	{"emote", "", "waves hello."},
	{"", "", ""},
	{"tell", "bob", ""},
}

func TestParseCommand(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn)

	world := NewWorld()

	bedroom, _ := world.NewRoom("The Bedroom")
	hall, _ := world.NewRoom("The Hall")
	den, _ := world.NewRoom("The Den")

	world.NewExit(bedroom, "west", hall)
	world.NewExit(bedroom, "say", den)    // Trying to be sneaky!

	player, _ := world.NewPlayer("bob", "foo", bedroom)
	client.player = player

	for i, cmd := range commandInputs {
		command := world.parseCommand(client, cmd)

		if command != expectedCommands[i] {
			t.Errorf("%d: Expected args to be equal. Actual: %s", i, command)
		}
	}
}

func TestPlayersCanBeAwakeOrAsleep(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)
	jim, _ := world.NewPlayer("jim", "foo", hall)

	if bob.awake || jim.awake {
		t.Errorf("Neither bob nor jim should be awake")
	}

	bob.awake = true

	if !bob.awake || jim.awake {
		t.Errorf("Bob should be awake, jim should not.")
	}

	jim.awake = true

	if !bob.awake || !jim.awake {
		t.Errorf("Bob and jim should be be awake.")
	}
}

func TestMovePlayer(t *testing.T) {
	// Changes room
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	den, _ := world.NewRoom("The Den")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	// Old room has bob in it
	if bob.location != hall {
		t.Errorf("Bob should be in the Hall")
	}

	if _, exists := hall.players[bob.key]; !exists {
		t.Errorf("Bob should be in the Hall")
	}

	if _, exists := den.players[bob.key]; exists {
		t.Errorf("Bob should not be in the Den")
	}

	world.MovePlayer(bob, den)

	// New room has bob in it.
	if bob.location != den {
		t.Errorf("Bob should be in the Den")
	}

	if _, exists := hall.players[bob.key]; exists {
		t.Errorf("Bob should not be in the set of Den players")
	}

	if _, exists := den.players[bob.key]; !exists {
		t.Errorf("Bob should be in the set of Den players")
	}

}

func TestDoConnectShouldWakeUpPlayers(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	if bob.awake {
		t.Errorf("Bob should not be awake.")
	}

	doConnect(world, client, Command{"connect", "bob", "foo"})

	if client.player != bob {
		t.Errorf("Connecting should have linked the client and the player")
	}

	if !bob.awake {
		t.Errorf("Connecting should have woken up bob.")
	}
}

func TestDoConnectDoesNothingIfPlayerNotFound(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "jim", "foo"})

	assertMatch(t, "No such player!\r\n", conn.String())

	if bob.awake {
		t.Errorf("Bob should still be asleep.")
	}
}

func TestDoConnectWithoutPasswordFAils(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "bob", ""})

	assertMatch(t, "Try: connect <player> <password>\r\n", conn.String())

	if bob.awake {
		t.Errorf("Bob should still be asleep.")
	}
}

func TestDoConnectWithWrongPasswordFAils(t *testing.T) {
	conn := NewMockConn()
	client := NewClient(conn) // &Client{conn: conn}
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	bob, _ := world.NewPlayer("bob", "foo", hall)

	doConnect(world, client, Command{"connect", "bob", "bar"})

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

	doConnect(world, bobClient, Command{"connect", "bob", "foo"})
	doConnect(world, jimClient, Command{"connect", "jim", "foo"})

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

	doConnect(world, bobClient, Command{"connect", "bob", "foo"})
	doConnect(world, jimClient, Command{"connect", "jim", "foo"})

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

	bobConn:= NewMockConn()
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

	doConnect(world, bobClient, Command{"connect", "bob", "foo"})
	doConnect(world, sallyClient, Command{"connect", "sally", "foo"})

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

func TestDoLookMeShowsMyDescription(t *testing.T) {
}
