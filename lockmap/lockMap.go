package lockmap

import "sync"

// A LockMap is used to lock various things represented by strings.
type LockMap struct {
	mutex sync.RWMutex
	m     map[string]*sync.Mutex
}

// New makes a new LockMap.
func New() *LockMap {
	return &LockMap{
		m: make(map[string]*sync.Mutex),
	}
}

func (lm *LockMap) getLock(str string) (mutex *sync.Mutex, found bool) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	mutex, found = lm.m[str]
	return mutex, found
}
func (lm *LockMap) addLock(str string) *sync.Mutex {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	mutex := &sync.Mutex{}
	lm.m[str] = mutex
	return mutex
}

// Lock locks the given str and returns a function to unlock the lock.
func (lm *LockMap) Lock(str string) (unlockFn func()) {
	mutex, found := lm.getLock(str)
	if !found {
		mutex = lm.addLock(str)
	}

	mutex.Lock()
	return func() {
		mutex.Unlock()
	}
}
