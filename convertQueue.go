package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func convertStreamFile(input string) error {
	dir, inputFile := filepath.Split(input)

	timestamp := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	output := filepath.Join(dir, timestamp+".mp4")

	log.Printf("starting converting %s to %s", input, output)

	if err := exec.Command(
		"ffmpeg",
		"-y",
		"-i",
		input,
		"-threads",
		"0",
		"-c:v",
		codec,
		"-s:v",
		resolution,
		output,
	).Run(); err != nil {
		return err
	}

	log.Printf("done converting %s to %s", input, output)

	return os.Remove(input)
}

type convertQueue chan string

func makeQueue(size int) convertQueue {
	ch := make(chan string, size)

	go func() {
		for path := range ch {
			if err := convertStreamFile(path); err != nil {
				panic(err)
			}
		}
	}()

	return ch
}
