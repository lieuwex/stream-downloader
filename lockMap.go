package main

import "sync"

type LockMap struct {
	mutex sync.RWMutex
	m     map[string]*sync.Mutex
}

func MakeLockMap() *LockMap {
	return &LockMap{
		m: make(map[string]*sync.Mutex),
	}
}

func (lm *LockMap) getLock(url string) (mutex *sync.Mutex, found bool) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	mutex, found = lm.m[url]
	return mutex, found
}
func (lm *LockMap) addLock(url string) *sync.Mutex {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	mutex := &sync.Mutex{}
	lm.m[url] = mutex
	return mutex
}

func (lm *LockMap) Lock(url string) (unlockFn func()) {
	mutex, found := lm.getLock(url)
	if !found {
		mutex = lm.addLock(url)
	}

	mutex.Lock()
	return func() {
		mutex.Unlock()
	}
}
