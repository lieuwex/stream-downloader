package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"stream-downloader/lockmap"
	"stream-downloader/streamlink"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	checkInterval = 30 * time.Second
	queueSize     = 50

	codec      = "libx265"
	resolution = "1280x720"
)

var (
	converterThreadCount = 0
	mainDir              string

	lm    = lockmap.New()
	queue = makeQueue(queueSize)
)

func getFolder(url string) (string, error) {
	ident := path.Base(url)

	folder := filepath.Join(mainDir, ident)
	if err := os.MkdirAll(folder, 0700); err != nil {
		return "", err
	}

	return folder, nil
}

func getOutputFile(url string) (string, error) {
	folder, err := getFolder(url)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s.ts", time.Now().Format("2006-01-02 15:04:05"))

	return filepath.Join(folder, fileName), nil
}

func handleStream(ctx context.Context, url string) {
	unlock := lm.Lock(url)
	defer unlock()

	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(checkInterval):
		}

		log.Printf("checking for %s\n", url)

		if online, err := streamlink.IsOnline(url); err != nil {
			log.Printf("error while checking if %s is online: %s\n", url, err)
			continue
		} else if !online {
			log.Printf("%s is offline\n", url)
			continue
		}

		outputFile, err := getOutputFile(url)
		if err != nil {
			log.Fatalf("error while creating folder for %s: %s\n", url, err)
			return
		}

		log.Printf("starting download for %s\n", url)

		cmd := streamlink.GetDownloadCommand(url, outputFile)
		if err := cmd.Run(); err != nil {
			log.Printf("error while running streamlink for %s: %s\n", url, err)
		} else {
			log.Printf("stream for %s successfully ended\n", url)
			queue <- outputFile
		}
	}
}

func parseStreamList(path string) ([]string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, err
	}

	list := strings.TrimSpace(string(bytes))
	return strings.Split(list, "\n"), nil
}

func main() {
	mainDir = os.Args[1]
	if val := os.Getenv("CONV_NUM_THREADS"); val != "" {
		var err error
		converterThreadCount, err = strconv.Atoi(val)
		if err != nil {
			log.Fatal(err)
		}
	}

	files, err := readDirRecursive(mainDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if filepath.Ext(f) != ".ts" {
			continue
		}

		queue <- f
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	listPath := mainDir + "streamlist"
	watcher.Add(listPath)

	for {
		ctx, cancel := context.WithCancel(context.Background())

		lines, err := parseStreamList(listPath)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("read %d stream(s) from stream list\n", len(lines))

		for _, url := range lines {
			go handleStream(ctx, url)
		}

		<-watcher.Events
		cancel()
	}
}
