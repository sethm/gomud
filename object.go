package main

import (
	"strings"
	"sync"
)

type Flags uint

// Flags
const (
	WizardFlag     Flags = 1 << iota
	BuilderFlag          = 1 << iota
	ProgrammerFlag       = 1 << iota
)

//
// Everything in the world is an Object.
//
type Object struct {
	sync.RWMutex
	// Every object in the database has a unique key. No two objects
	// have the same key.
	key int
	// Everything has a displayable name.
	name string
	// Everything has a normalized name that's used for comparisons
	// and lookups where case does not matter.
	normalName string
	// Everything has a textual description.
	description string
	// Everything has an owner. Since players are Objects, this means
	// that even Players have owners, but we do not condone slavery
	// here so this particular field is completely ignored on Players.
	owner *Player

	// Player flags
	flags Flags
}

//
// All objects must satisfy the Objecter interface
//
type Objecter interface {
	Key() int
	Name() string
	SetName(s string)
	NormalName() string
	Description() string
	SetDescription(s string)
	Owner() *Player
	SetOwner(p *Player)
	SetFlag(f Flags)
	ClearFlag(f Flags)
	IsSet(f Flags) bool
}

//
// Object implements the Objecter
// interface
//
func (o *Object) Key() int {
	return o.key
}

func (o *Object) Name() string {
	return o.name
}

func (o *Object) SetName(s string) {
	o.name = s
	o.normalName = strings.ToLower(s)
}

func (o *Object) NormalName() string {
	return o.normalName
}

func (o *Object) Description() string {
	if o.description == "" {
		return "You see nothing special."
	}

	return o.description
}

func (o *Object) SetDescription(s string) {
	o.description = s
}

func (o *Object) Owner() *Player {
	return o.owner
}

func (o *Object) SetOwner(p *Player) {
	o.owner = p
}

func (o *Object) SetFlag(f Flags) {
	o.flags = (f | BuilderFlag)
}

func (o *Object) ClearFlag(f Flags) {
	o.flags ^= (f | BuilderFlag)
}

func (o *Object) IsSet(f Flags) bool {
	return (o.flags & f) != 0
}
