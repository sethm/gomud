package main

import "crypto/sha512"

//
// A player interacts with the world
//
type Player struct {
	Object
	password [64]byte
	location *Room
	awake    bool
	client   *Client
}

func (p *Player) SetPassword(raw string) {
	p.password = sha512.Sum512([]byte(raw))
}

func (p *Player) CanSetFlag(target Objecter, flag Flags) bool {
	switch flag {
	default:
		return false // Default is resricted
	case WizardFlag:
		switch target.(type) {
		default:
			return false
		case *Player:
			return p.IsSet(WizardFlag)
		}
	case BuilderFlag:
		switch target.(type) {
		default:
			return false
		case *Player:
			return p.IsSet(WizardFlag)
		}
	}
}
