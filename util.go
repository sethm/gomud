package main

//
// Various utility functions used by handlers, etc.
//

func hasBuildPermission(p *Player) bool {
	return p.IsSet(WizardFlag) || p.IsSet(BuilderFlag)
}
