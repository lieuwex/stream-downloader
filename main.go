package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"stream-downloader/chat"
	"stream-downloader/lockmap"
	"stream-downloader/streamlink"
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

func handleStream(ctx context.Context, chatClient *chat.Client, url string) {
	unlock := lm.Lock(url)
	defer unlock()

	twitchUsername := strings.TrimPrefix(url, "https://www.twitch.tv/")
	if twitchUsername == url {
		// stream url is not a twitch url
		twitchUsername = ""
	}
	if chatClient == nil {
		twitchUsername = ""
	}

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

		log.Printf("starting download for %s (twitchUsername = %s)\n", url, twitchUsername)

		var f *os.File
		if twitchUsername != "" {
			var err error
			f, err = os.Create(strings.Replace(outputFile, ".ts", ".txt", 1))
			if err != nil {
				log.Printf("error while create chat output file: %s", err)
			} else {
				encoder := json.NewEncoder(f)

				chatClient.AddChatFunction(twitchUsername, func(msg chat.Message) {
					encoder.Encode(msg)
				})
			}
		}

		cmd := streamlink.GetDownloadCommand(url, outputFile)
		if err := cmd.Run(); err != nil {
			log.Printf("error while running streamlink for %s: %s\n", url, err)
		}

		log.Printf("stream for %s ended\n", url)
		queue <- convertItem{outputFile, 0}

		if f != nil {
			chatClient.RemoveChatFunction(twitchUsername)
			f.Close()
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

	var chatClient *chat.Client
	if val := os.Getenv("TWITCH_AUTH"); val != "" {
		splitted := strings.SplitN(val, ":", 2)
		username := splitted[0]
		apiKey := splitted[1]

		log.Printf("created chat client for username %s", username)

		chatClient = chat.CreateClient()
		go func() {
			err := chatClient.Connect(username, apiKey)
			if err != nil {
				log.Printf("error connecting to twitch irc: %s", err)
			}
		}()
	}

	files, err := readDirRecursive(mainDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if filepath.Ext(f) != ".ts" {
			continue
		}

		queue <- convertItem{f, 0}
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
			go handleStream(ctx, chatClient, url)
		}

		<-watcher.Events
		cancel()
	}
}
