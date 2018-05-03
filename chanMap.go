package main

import "sync"

type ChanMap struct {
	mutex sync.RWMutex
	m     map[string]chan bool
}

func makeChanMap() *ChanMap {
	return &ChanMap{
		m: make(map[string]chan bool),
	}
}

func (ch *ChanMap) GetChannel(url string) (chan bool, bool) {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()

	val, found := ch.m[url]
	return val, found
}
func (ch *ChanMap) AddDownloading(url string) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	ch.m[url] = make(chan bool)
}
func (ch *ChanMap) RemoveDownloading(url string) {
	channel, found := ch.GetChannel(url)
	if !found {
		return
	}

	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	close(channel)
	ch.m[url] = nil
}
