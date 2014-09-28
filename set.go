package main

import "container/list"

type Object interface {
	// Return the object's key.
	Key() int
	// Return the name of an item.
	Name() string
	// Return the description of an item.
	Description() string
}

type Selector func(o Object) bool

type Set struct {
	container map[int]Object
}

func NewSet() Set {
	return Set{make(map[int]Object)}
}

func (s Set) Iterator() <-chan Object {
	ch := make(chan Object)
	go func() {
		for _, val := range s.container {
			ch<- val
		}
		close(ch)
	}()
	return ch
}

func (s Set) Add(o Object) bool {
	if s.container[o.Key()] != nil {
		return false
	}

	s.container[o.Key()] = o
	return true
}

func (s Set) Contains(o Object) bool {	
	_, exists := s.container[o.Key()]
	return exists
}

func (s Set) Remove(o Object) {
	delete(s.container, o.Key())
}

func (s Set) Select(f Selector) []Object {
	l := list.New()

	// Build a list of selected objects
	for _,v := range s.container {
		if f(v) {
			l.PushBack(v)
		}
	}

	// Convert the list into an array.
	arr := make([]Object, l.Len(), l.Len())
	i := 0
	
	for e := l.Front(); e != nil; e = e.Next() {
		arr[i] = e.Value.(Object)
		i++
	}

	return arr
}

func (s Set) SelectOne(f Selector) Object {
	for _,v := range s.container {
		if f(v) {
			return v
		}
	}

	return nil
}

func (s Set) ContainsWhere(f Selector) bool {
	for _,v := range s.container {
		if f(v) {
			return true
		}
	}

	return false
}

func (s Set) Len() int {
	return len(s.container)
}
