package server

import "sort"

type Roundrobin struct {
	lock  chan bool
	names []string
	index int
}

func (r *Roundrobin) Next() []string {
	if len(r.names) == 0 {
		return make([]string, 0)
	}

	r.lock <- true
	defer func() {
		<-r.lock
	}()

	r.index++
	if r.index > len(r.names)-1 {
		r.index = 0
	}

	if r.index == 0 {
		return r.names
	}

	return append(r.names[r.index:], r.names[:r.index]...)
}

// NewRoundrobin creates a new Roundrobin selector.
func NewRoundrobin(providers []string) *Roundrobin {
	names := append(make([]string, 0, len(providers)), providers...)

	sort.Strings(names)

	return &Roundrobin{
		lock:  make(chan bool, 1),
		names: names,
		index: -1,
	}
}
