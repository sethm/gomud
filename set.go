package main

import "container/list"

type Object interface {
	// Return the name of an item.
	Name() string
	// Return the description of an item.
	Description() string
}

type Finder func(o Object) bool

type Set struct {
	container map[Object]bool
}

func NewSet() Set {
	return Set{make(map[Object]bool)}
}

func (s Set) Add(i Object) bool {
	if s.container[i] {
		return false
	}

	s.container[i] = true
	return true
}

func (s Set) Contains(i Object) bool {	
	return s.container[i] == true
}

func (s Set) Remove(i Object) {
	delete(s.container, i)
}

func (s Set) Find(f Finder) []Object {

	l := list.New()
	
	for k,_ := range s.container {
		if f(k) {
			l.PushBack(k)
		}
	}

	arr := make([]Object, l.Len(), l.Len())
	i := 0
	
	for e := l.Front(); e != nil; e = e.Next() {
		arr[i] = e.Value.(Object)
		i++
	}

	return arr
}

func (s Set) ContainsWhere(f Finder) bool {
	for k,_ := range s.container {
		if f(k) {
			return true
		}
	}

	return false
}
