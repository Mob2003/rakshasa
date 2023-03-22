package server

//封装一下易于调试的lock
import (
	"fmt"
	"rakshasa/common"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
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
	if common.DebugLock {

		_, file, line, _ := runtime.Caller(1)
		key := file + "行" + strconv.Itoa(line)
		u.key = key

		var n *int32
		if v, ok := common.DebugLockMap.Load(key); ok {
			n = v.(*int32)
		} else {
			a := int32(0)
			n = &a
			common.DebugLockMap.Store(key, n)
		}
		atomic.AddInt32(n, 1)
	}

	l.l.Lock()
	return u
}
func (l *lock) RLock(old ...*unlock) *unlock {
	u := &unlock{l: &l.l}
	if len(old) == 1 {
		u = old[0]
	}
	if common.DebugLock {

		_, file, line, _ := runtime.Caller(1)
		key := file + "行" + strconv.Itoa(line)
		u.key = key
		var n *int32
		if v, ok := common.DebugLockMap.Load(key); ok {
			n = v.(*int32)
		} else {
			a := int32(0)
			n = &a
			common.DebugLockMap.Store(key, n)
		}
		atomic.AddInt32(n, 1)
	}

	l.l.RLock()
	return u
}
func (l *unlock) Unlock() {
	if common.DebugLock {
		if v, ok := common.DebugLockMap.Load(l.key); ok {
			atomic.AddInt32(v.(*int32), -1)
		} else {
			panic("")
		}
	}
	l.l.Unlock()
}
func (l *unlock) RUnlock() {
	if common.DebugLock {
		if v, ok := common.DebugLockMap.Load(l.key); ok {
			atomic.AddInt32(v.(*int32), -1)
		} else {
			panic("")
		}
	}
	l.l.RUnlock()
}
func printLock() {
	common.DebugLockMap.Range(func(key, value interface{}) bool {
		fmt.Println(key, *value.(*int32))
		return true
	})
}
