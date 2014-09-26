package main

import "testing"

func TestNewSet(t *testing.T) {
	NewSet()
}

type TestObject struct {
	name, description string
}

func (o TestObject) Name() string {
	return o.name
}

func (o TestObject) Description() string {
	return o.description
}

func TestAdd(t *testing.T) {
	s := NewSet()

	if !s.Add(&TestObject{"bob", "Bob is tall"}) {
		t.Errorf("Should have been able to add bob")
	}
}

func TestAddCannotAddDuplicates(t *testing.T) {
 	s := NewSet()

 	s.Add(&TestObject{"bob", "Bob is tall"})
	
 	if s.Add(&TestObject{"bob", "Bob is tall"}) {
 		t.Errorf("Should not be able to add duplicates to the set.")
 	}
 }

// func TestContains(t *testing.T) {
// 	s := NewSet()

// 	if s.Contains(5) {
// 		t.Errorf("Should not contain '5'")
// 	}

// 	s.Add(5)

// 	if !s.Contains(5) {
// 		t.Errorf("Should contain '5'")
// 	}
// }

// func TestRemoveRemovesItem(t *testing.T) {
// 	s := NewSet()
// 	s.Add(5)
// 	s.Add(202)

// 	if !s.Contains(5) || !s.Contains(202) {
// 		t.Errorf("Set is not in expected state.")
// 	}

// 	s.Remove(5)

// 	if s.Contains(5) || !s.Contains(202) {
// 		t.Errorf("Set is not in expected state.")
// 	}

// 	s.Remove(202)

// 	if s.Contains(5) || s.Contains(202) {
// 		t.Errorf("Set is not in expected state.")
// 	}

// 	// Should cause no errors
// 	s.Remove(1234)
// }

// func TestFind(t *testing.T) {
// 	s := NewSet()
// 	s.Add(5)
// 	s.Add(8)
// 	s.Add(15)

// 	fiveFinder := func (i interface{}) bool {
// 		return i.(int) == 5
// 	}

// 	fifteenFinder := func (i interface{}) bool {
// 		return i.(int) % 5 == 0 && i.(int) % 3 == 0
// 	}

// 	twelveFinder := func (i interface{}) bool {
// 		return i.(int) == 12
// 	}

// 	divisibleByFiveFinder := func (i interface{}) bool {
// 		return i.(int) % 5 == 0
// 	}

// 	i := s.Find(fiveFinder)
// 	j := s.Find(fifteenFinder)
// 	k := s.Find(twelveFinder)
// 	l := s.Find(divisibleByFiveFinder)

// 	if len(i) != 1 || i[0] != 5 {
// 		t.Errorf("Should have found '5'")
// 	}

// 	if len(j) != 1 || j[0] != 15 {
// 		t.Errorf("Should have found '15'")
// 	}

// 	if len(k) != 0 {
// 		t.Errorf("Should not have found '12'")
// 	}

// 	if len(l) != 2 ||
// 		(l[0] != 5 && l[1] != 5) ||
// 		(l[0] != 15 && l[1] != 15) {
// 		t.Errorf("Should have found '5' and '15'")
// 	}
// }

// func TestContainsWhere(t *testing.T) {
// 	s := NewSet()

// 	s.Add(5)
// 	s.Add(8)
// 	s.Add(15)

// 	fiveFinder := func (i interface{}) bool {
// 		return i.(int) == 5
// 	}

// 	fifteenFinder := func (i interface{}) bool {
// 		return i.(int) % 5 == 0 && i.(int) % 3 == 0
// 	}

// 	twelveFinder := func (i interface{}) bool {
// 		return i.(int) == 12
// 	}

// 	divisibleByFiveFinder := func (i interface{}) bool {
// 		return i.(int) % 5 == 0
// 	}

// 	if !s.ContainsWhere(fiveFinder) {
// 		t.Errorf("Should have returned true for ContainsWhere")
// 	}

// 	if !s.ContainsWhere(fifteenFinder) {
// 		t.Errorf("Should have returned true for ContainsWhere")
// 	}

// 	if s.ContainsWhere(twelveFinder) {
// 		t.Errorf("Should have returned false for ContainsWhere")
// 	}

// 	if !s.ContainsWhere(divisibleByFiveFinder) {
// 		t.Errorf("Should have returned true for ContainsWhere")
// 	}
// }
