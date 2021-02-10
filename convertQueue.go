package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const maxTryCount = 5

func convertStreamFile(input string) error {
	dir, inputFile := filepath.Split(input)

	timestamp := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	output := filepath.Join(dir, timestamp+".mp4")

	log.Printf("starting converting %s to %s", input, output)

	if err := exec.Command(
		"nice",
		"-n",
		"19",

		"ffmpeg",
		"-y",
		"-i",
		input,
		"-threads",
		strconv.Itoa(converterThreadCount),
		"-c:v",
		codec,
		"-s:v",
		resolution,
		output,
	).Run(); err != nil {
		return err
	}

	log.Printf("done converting %s to %s", input, output)

	// TODO: touch -r

	if err := os.Remove(input); err != nil {
		log.Printf("error while removing input file %s: %s", input, err)
	}

	return nil
}

type convertItem struct {
	Path     string
	TryCount uint
}
type convertQueue chan convertItem

func makeQueue(size int) convertQueue {
	ch := make(chan convertItem, size)

	go func() {
		for item := range ch {
			if item.TryCount == maxTryCount {
				log.Printf("too much tries for %s", item.Path)
				continue
			}

			if err := convertStreamFile(item.Path); err != nil {
				log.Printf("error while converting %s: %s, trying again", item.Path, err)
				ch <- convertItem{item.Path, item.TryCount + 1}
			}
		}
	}()

	return ch
}
