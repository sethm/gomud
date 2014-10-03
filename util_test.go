package main

import (
	"testing"
)

func TestHasPermissionRespectsBuilderFlag(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	jim, _ := world.NewPlayer("jim", "foo", hall)

	if hasBuildPermission(jim) {
		t.Errorf("Jim should not have build permission.")
	}

	jim.SetFlag(BuilderFlag)

	if !hasBuildPermission(jim) {
		t.Errorf("Jim should have build permission.")
	}
}

func TestHasPermissionRespectsWizardFlag(t *testing.T) {
	world := NewWorld()
	hall, _ := world.NewRoom("The Hall")
	jim, _ := world.NewPlayer("jim", "foo", hall)

	if hasBuildPermission(jim) {
		t.Errorf("Jim should not have build permission.")
	}

	jim.SetFlag(WizardFlag)

	if !hasBuildPermission(jim) {
		t.Errorf("Jim should have build permission.")
	}
}
