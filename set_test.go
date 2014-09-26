package main

import "testing"
import "regexp"

type TestObject struct {
	key int
	name, description string
}

func (o TestObject) Key() int {
	return o.key
}

func (o TestObject) Name() string {
	return o.name
}

func (o TestObject) Description() string {
	return o.description
}

func TestNewSet(t *testing.T) {
	NewSet()
}

func TestAdd(t *testing.T) {
	s := NewSet()

	if !s.Add(&TestObject{0, "bob", "Bob is tall"}) {
		t.Errorf("Should have been able to add bob")
	}
}

func TestAddCannotAddDuplicates(t *testing.T) {
 	s := NewSet()

 	s.Add(&TestObject{1, "bob", "Bob is tall"})
	
 	if s.Add(&TestObject{1, "bob", "Bob is tall"}) {
 		t.Errorf("Should not be able to add duplicates to the set.")
 	}
}

func TestContains(t *testing.T) {
	s := NewSet()
	o := &TestObject{1, "bob", "Bob is tall"}

	if s.Contains(o) {
		t.Errorf("Should not contain bob")
	}

	s.Add(o)

	if !s.Contains(o) {
		t.Errorf("Should contain bob")
	}
}

func TestRemoveRemovesItem(t *testing.T) {
	s := NewSet()
	bob := &TestObject{0, "bob", "Bob is tall."}
	bill := &TestObject{1, "bill", "Bill is short."}

	s.Add(bob)
	s.Add(bill)

	if !s.Contains(bob) || !s.Contains(bill) {
		t.Errorf("Set is not in expected state.")
	}

	s.Remove(bob)

	if s.Contains(bob) || !s.Contains(bill) {
		t.Errorf("Set is not in expected state.")
	}

	s.Remove(bill)

	if s.Contains(bob) || s.Contains(bill) {
		t.Errorf("Set is not in expected state.")
	}

	// Should cause no errors
	s.Remove(&TestObject{2, "jane", "Jane was never in the set."})
}

func TestSelectFirst(t *testing.T) {
	s := NewSet()

	bob := &TestObject{0, "bob", "Bob is tall."}
	bill := &TestObject{1, "bill", "Bill is short."}
	jane := &TestObject{2, "jane", "Jane is tall, too."}

	s.Add(bob)
	s.Add(bill)
	s.Add(jane)

	bobSelector := func (o Object) bool {
		return o.Name() == "bob"
	}

	billSelector := func (o Object) bool {
		return o.Name() == "bill"
	}

	georgeSelector := func (o Object) bool {
		return o.Name() == "george"
	}

	tallSelector := func (o Object) bool {
		match, _ := regexp.MatchString("tall", o.Description())
		return match
	}

	i := s.SelectFirst(bobSelector)
	j := s.SelectFirst(billSelector)
	k := s.SelectFirst(georgeSelector)
	l := s.SelectFirst(tallSelector)

	if i != bob {
		t.Errorf("Should have selected Bob")
	}

	if j != bill {
		t.Errorf("Should have selected Bill")
	}

	if k != nil {
		t.Errorf("Should not have selected anyone")
	}

	if l != bob && l != jane {
		t.Errorf("Should have selected Bob or Jane")
	}
}

func TestSelect(t *testing.T) {
	s := NewSet()

	bob := &TestObject{0, "bob", "Bob is tall."}
	bill := &TestObject{1, "bill", "Bill is short."}
	jane := &TestObject{2, "jane", "Jane is tall, too."}

	s.Add(bob)
	s.Add(bill)
	s.Add(jane)

	bobSelector := func (o Object) bool {
		return o.Name() == "bob"
	}

	billSelector := func (o Object) bool {
		return o.Name() == "bill"
	}

	georgeSelector := func (o Object) bool {
		return o.Name() == "george"
	}

	tallSelector := func (o Object) bool {
		match, _ := regexp.MatchString("tall", o.Description())
		return match
	}

	i := s.Select(bobSelector)
	j := s.Select(billSelector)
	k := s.Select(georgeSelector)
	l := s.Select(tallSelector)

	if len(i) != 1 || i[0] != bob {
		t.Errorf("Should have found bob")
	}

	if len(j) != 1 || j[0] != bill {
		t.Errorf("Should have found bill ")
	}

	if len(k) != 0 {
		t.Errorf("Should not have found george")
	}

	if len(l) != 2 ||
		(l[0] != bob && l[1] != bob) ||
		(l[0] != jane && l[1] != jane) {
		t.Errorf("Should have found bob and jane")
	}
}

func TestContainsWhere(t *testing.T) {
	s := NewSet()

	bob := &TestObject{0, "bob", "Bob is tall."}
	bill := &TestObject{1, "bill", "Bill is short."}
	jane := &TestObject{2, "jane", "Jane is tall, too."}

	s.Add(bob)
	s.Add(bill)
	s.Add(jane)

	bobSelector := func (o Object) bool {
		return o.Name() == "bob"
	}

	billSelector := func (o Object) bool {
		return o.Name() == "bill"
	}

	georgeSelector := func (o Object) bool {
		return o.Name() == "george"
	}

	tallSelector := func (o Object) bool {
		match, _ := regexp.MatchString("tall", o.Description())
		return match
	}

	if !s.ContainsWhere(bobSelector) {
		t.Errorf("Should have found bob")		
	}

	if !s.ContainsWhere(billSelector) {
		t.Errorf("Should have found bill")
	}

	if s.ContainsWhere(georgeSelector) {
		t.Errorf("Should not have found george")
	}

	if !s.ContainsWhere(tallSelector) {
		t.Errorf("Should have found bill and jane")
	}
}
