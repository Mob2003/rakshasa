package server

//封装一下易于调试的lock
import (
	"sync"
)

type lock struct {
	l sync.RWMutex
}
type unlock struct {
	key string
	l   *sync.RWMutex
}

func (l *lock) Lock(old ...*unlock) *unlock {
	u := &unlock{l: &l.l}
	if len(old) == 1 {
		u = old[0]
	}
	l.l.Lock()
	return u
}
func (l *lock) RLock(old ...*unlock) *unlock {
	u := &unlock{l: &l.l}
	if len(old) == 1 {
		u = old[0]
	}

	l.l.RLock()
	return u
}
func (l *unlock) Unlock() {
	l.l.Unlock()
}
func (l *unlock) RUnlock() {
	l.l.RUnlock()
}
