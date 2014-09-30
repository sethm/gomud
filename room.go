package main

//
// Exits link two rooms together
//
type Exit struct {
	Object
	Ownable
	destination *Room
}

//
// A room is a place in the world.
//
type Room struct {
	Object
	Ownable
	exits       map[int]*Exit
	players     map[int]*Player
}

