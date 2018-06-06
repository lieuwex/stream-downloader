package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"stream-downloader/lockmap"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	checkInterval = 30 * time.Second
	queueSize     = 50

	codec      = "libx264"
	resolution = "1280x720"
)

var (
	mainDir string
	lm      = lockmap.New()
	queue   = makeQueue(queueSize)
)

func getFolder(url string) (string, error) {
	ident := path.Base(url)

	folder := filepath.Join(mainDir, ident)
	if err := os.MkdirAll(folder, 0700); err != nil && !os.IsExist(err) {
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

		outputFile, err := getOutputFile(url)
		if err != nil {
			log.Fatalf("error while creating folder for %s: %s\n", url, err.Error())
			return
		}

		streamlinkCmd := exec.Command(
			"streamlink",
			"--twitch-disable-hosting",
			url,
			"1080p,720p,best",
			"-o",
			outputFile,
		)

		if err := streamlinkCmd.Run(); err == nil {
			log.Printf("stream for %s ended\n", url)
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
