package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

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

	if err := os.Remove(input); err != nil {
		log.Printf("error while removing input file %s: %s", input, err)
	}

	return nil
}

type convertQueue chan string

func makeQueue(size int) convertQueue {
	ch := make(chan string, size)

	go func() {
		for path := range ch {
			if err := convertStreamFile(path); err != nil {
				log.Printf("error while converting %s: %s, trying again.", path, err)
				ch <- path
			}
		}
	}()

	return ch
}
