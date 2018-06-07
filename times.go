package main

import (
	"os"

	"github.com/lieuwex/times"
)

func getTimes(name string) (times.Timespec, error) {
	return times.Stat(name)
}

func setTimes(name string, timespec times.Timespec) error {
	atime, _ := timespec.AccessTime()
	mtime := timespec.ModTime()
	return os.Chtimes(name, atime, mtime)
}
