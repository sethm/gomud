package main

import "time"
import "net"
import "testing"

//
// Implement a mock version of net.Conn for testing
//

type MockConn struct {
	readBytes [][]byte
	writtenBytes []byte

	readError *error
	closeAfterWrites int
	numWrites int
	writtenBytePtr int
}

func (conn MockConn) Read(b []byte) (n int, err error) {
	read := copy(b, conn.readBytes[conn.numWrites])
	conn.numWrites++
	return read, nil
}

func (conn MockConn) Write(b []byte) (n int, err error) {
	written := copy(conn.writtenBytes[conn.writtenBytePtr:], b)
	conn.writtenBytePtr += written
	return written, nil
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
	return &MockConn{readBytes: make([][]byte, 1024, 1024),	writtenBytes: make([]byte, 1024, 1024)}
}

// Create a new world, with no players.

func TestNewWorld(t *testing.T) {
	w := NewWorld()

	if len(w.players) != 0 {
		t.Errorf("Expected world to exist and have 0 players.")
	}
}

// Create a room in the world.
func TestNewRoom(t *testing.T) {
	world := NewWorld()
	hall, err := world.NewRoom("The Hall")
	if hall == nil || err != nil {	
		t.Errorf("Expected to create a new room.")
	}
	if len(world.rooms) != 1 || !world.rooms[hall] {
		t.Errorf("Expected room to have been added to the world.")
	}
}

// Create a player in the world.
func TestNewPlayer(t *testing.T) {
	world := NewWorld()
	hall,_ := world.NewRoom("The Hall")
	bob,_ := world.NewPlayer("bob", hall)
	if bob.name != "bob" {
		t.Errorf("Expected player name to be bob, but was %s", bob.name)
	}
	if bob.location != hall {
		t.Errorf("Expected player's location to be the hall")
	}
	if len(world.players) != 1 || world.players["bob"] != bob {
		t.Errorf("Expected player to have been added to world.")
	}
}

func TestNewPlayerCantReuseNames(t *testing.T) {
	world := NewWorld()
	hall,_ := world.NewRoom("The Hall")
	world.NewPlayer("bob", hall)
	otherBob, err := world.NewPlayer("bob", hall)
	if otherBob != nil || err == nil {
		t.Errorf("Should not have been able to create duplicate user")
	}
}

func TestConnectPlayerSucceedsWhenPlayerFound(t *testing.T) {
	world := NewWorld()
	hall,_ := world.NewRoom("The Hall")
	conn := NewMockConn()
	world.NewPlayer("bob", hall)
	bob, err := world.ConnectPlayer("bob", conn)

	if (err != nil) {
		t.Errorf("Could not find player:", err)
	}

	if (bob.conn != conn) {
		t.Errorf("Should have connected the user.")
	}
}

func TestConnectPlayerFailsWhenPlayerNotFound(t *testing.T) {
	world := NewWorld()
	conn := NewMockConn()
	bob, err := world.ConnectPlayer("bob", conn)

	if (bob != nil || err == nil) {
		t.Errorf("Expected player not to be found.")
	}
}

func TestDisconnectPlayerSucceedsWhenPlayerFound(t *testing.T) {
	world := NewWorld()
	hall,_ := world.NewRoom("The Hall")
	conn := NewMockConn()
	world.NewPlayer("bob", hall)
	bob, err := world.ConnectPlayer("bob", conn)

	if (err != nil) {
		t.Errorf("Could not find player:", err)
	}
	
	world.DisconnectPlayer("bob")
	
	if (bob.conn != nil) {
		t.Errorf("Should have disconnected the user.")
	}
}

func TestDisconnectPlayerFailsWhenPlayerNotFound(t *testing.T) {
	world := NewWorld()
	bob, err := world.DisconnectPlayer("bob")
	
	if (bob != nil || err == nil) {
		t.Errorf("Expected player not to be found.")
	}
}

func TestTell(t *testing.T) {
	world := NewWorld()
	hall,_ := world.NewRoom("The Hall")
	conn := NewMockConn()
	world.NewPlayer("bob", hall)
	bob, err := world.ConnectPlayer("bob", conn)

	if (err != nil) {
		t.Errorf("Could not find player:", err)
	}

	bob.tell("Hello, world!\n")

	actual := string(conn.writtenBytes[0:15])
	if actual != "Hello, world!\n\u0000" {
		t.Errorf("`tell` did not write bytes correctly: '%s'", actual)
	}
}
