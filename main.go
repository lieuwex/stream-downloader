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
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	mainDir  = "/media/lieuwe/streams/"
	listPath = mainDir + "streamlist"

	checkInterval = 30 * time.Second
)

var ch = makeChanMap()

func getOutputFile(url string) (string, error) {
	ident := path.Base(url)

	folder := filepath.Join(mainDir, ident)
	if err := os.MkdirAll(folder, 0700); err != nil && !os.IsExist(err) {
		return "", err
	}

	fileName := fmt.Sprintf("%d.mp4", time.Now().Unix())

	return filepath.Join(folder, fileName), nil
}

func handleStream(ctx context.Context, url string) {
	if ch, found := ch.GetChannel(url); found {
		<-ch
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(checkInterval):
		}

		log.Printf("checking for %s\n", url)

		outputFile, err := getOutputFile(url)
		if err != nil {
			log.Printf("error while creating folder for %s: %s\n", url, err.Error())
			return
		}

		ch.AddDownloading(url)
		cmd := exec.Command("streamlink", url, "1080p,720p,best", "-o", outputFile)
		if err := cmd.Run(); err == nil {
			log.Printf("stream for %s ended\n", url)
		}
		ch.RemoveDownloading(url)
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
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

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
