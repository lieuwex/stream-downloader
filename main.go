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
	"runtime"
	"strconv"
	"stream-downloader/lockmap"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const checkInterval = 30 * time.Second

var (
	mainDir string
	lm      = lockmap.New()
)

func getListPath() string {
	return mainDir + "streamlist"
}

func getOutputFile(time time.Time, url string, tmp bool) (string, error) {
	ident := path.Base(url)

	folder := filepath.Join(mainDir, ident)
	if tmp {
		folder = filepath.Join(folder, "tmp")
	}

	if err := os.MkdirAll(folder, 0700); err != nil && !os.IsExist(err) {
		return "", err
	}

	var ext string
	if tmp {
		ext = "ts"
	} else {
		ext = "mp4"
	}

	fileName := fmt.Sprintf("%d.%s", time.Unix(), ext)

	return filepath.Join(folder, fileName), nil
}

func convertStream(time time.Time, url string) error {
	inputFile, err := getOutputFile(time, url, true)
	if err != nil {
		return err
	}

	outputFile, err := getOutputFile(time, url, false)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"ffmpeg",
		"-i",
		inputFile,
		"-c:v",
		"libx264",
		"-threads",
		strconv.Itoa(runtime.NumCPU()),
		outputFile,
	)

	if err := cmd.Run(); err != nil {
		return err
	}

	return os.Remove(inputFile)
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

		time := time.Now()
		outputFile, err := getOutputFile(time, url, true)
		if err != nil {
			log.Fatalf("error while creating folder for %s: %s\n", url, err.Error())
			return
		}

		cmd := exec.Command(
			"streamlink",
			"--twitch-disable-hosting",
			url,
			"1080p,720p,best",
			"-o",
			outputFile,
		)
		if err := cmd.Run(); err == nil {
			log.Printf("stream for %s ended\n", url)
		}

		go func() {
			if err := convertStream(time, url); err != nil {
				log.Printf("error while converting stream %s: %s\n", outputFile, err)
				return
			}

			log.Printf("done converting stream %s\n", outputFile)
		}()
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

	mainDir = os.Args[1]

	listPath := getListPath()
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
