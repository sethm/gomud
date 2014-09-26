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

// Create a player in the world.
func TestNewPlayer(t *testing.T) {
	world := NewWorld()
	bob,_ := world.NewPlayer("bob")

	if bob.name != "bob" {
		t.Errorf("Expected player name to be bob, but was %s", bob.name)
	}
	
}

func TestNewPlayerCantReuseNames(t *testing.T) {
	world := NewWorld()
	bob, err := world.NewPlayer("bob")
	if bob == nil {
		t.Errorf("User should not have been nil")
	}

	if err != nil {
		t.Errorf("Should have been no error creating the user")
	}

	otherBob, err := world.NewPlayer("bob")
	if otherBob != nil {
		t.Errorf("User should have been nil")
	}

	if err == nil {
		t.Errorf("An error was expected")
	}
	
	
}

func TestConnectPlayerSucceedsWhenPlayerFound(t *testing.T) {
	world := NewWorld()
	conn := NewMockConn()
	world.NewPlayer("bob")
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
	conn := NewMockConn()
	world.NewPlayer("bob")
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
	conn := NewMockConn()
	world.NewPlayer("bob")
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
