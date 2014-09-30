package main

import (
	"sync"
	"strings"
)

//
// Everything in the world is an Object
//
type Object struct {
	sync.RWMutex
	key         int
	name        string
	normalName  string
	description string
	flags       Flags
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

	Flags() Flags
	SetFlags(f Flags)
}

//
// Some objects are also ownable.
//
type Ownable struct {
	owner       *Player
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

func (o *Object) Flags() Flags {
	return o.flags
}

func (o *Object) SetFlags(f Flags) {
	o.flags = f
}
