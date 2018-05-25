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
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	checkInterval = 30 * time.Second

	codec      = "libx264"
	resolution = "1280x720"
)

var (
	mainDir string
	lm      = lockmap.New()
)

func getListPath() string {
	return mainDir + "streamlist"
}

func getOutputFile(url string) (string, error) {
	ident := path.Base(url)

	folder := filepath.Join(mainDir, ident)
	if err := os.MkdirAll(folder, 0700); err != nil && !os.IsExist(err) {
		return "", err
	}

	fileName := fmt.Sprintf("%s.mp4", time.Now().Format("2006-01-02 15:04:05"))

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

		checkCmd := exec.Command(
			"streamlink",
			"--twitch-disable-hosting",
			url,
		)
		checkCmd.Start()
		state, _ := checkCmd.Process.Wait()
		waitStatus := state.Sys().(syscall.WaitStatus)
		if waitStatus.ExitStatus() != 0 {
			// stream not live
			continue
		}

		streamlinkCmd := exec.Command(
			"streamlink",
			"--twitch-disable-hosting",
			url,
			"1080p,720p,best",
			"-O",
		)
		ffmpegCmd := exec.Command(
			"ffmpeg",
			"-i",
			"pipe:0",
			"-threads",
			"0",
			"-c:v",
			codec,
			"-s:v",
			resolution,
			outputFile,
		)

		ffmpegCmd.Stdin, _ = streamlinkCmd.StdoutPipe()
		ffmpegCmd.Start()

		if err := streamlinkCmd.Run(); err == nil {
			log.Printf("stream for %s ended\n", url)
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
