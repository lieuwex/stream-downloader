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
	"stream-downloader/chat"
	"stream-downloader/convert"
	"stream-downloader/lockmap"
	"stream-downloader/streamlink"
	"stream-downloader/twitch"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

const (
	checkInterval = 30 * time.Second
)

var (
	mainDir  string
	clientId string

	lm    = lockmap.New()
	queue = convert.MakeQueue(convert.Settings{
		Size: 50,

		MaxVideoWidth:  1920,
		MaxVideoHeight: 1080,
	})
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

func writeYamlFile(videoPath string, info *StreamInfo) error {
	dir, inputFile := filepath.Split(videoPath)

	timestamp := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	filepath := filepath.Join(dir, timestamp+".yaml")

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(file)
	defer enc.Close()

	return enc.Encode(info)
}

type DatapointGatherer struct {
	ctx    context.Context
	mu     sync.Mutex
	cancel context.CancelFunc
	info   StreamInfo
}

func NewDatapointGatherer() *DatapointGatherer {
	ctx, cancel := context.WithCancel(context.Background())
	return &DatapointGatherer{ctx: ctx, cancel: cancel}
}
func (c *DatapointGatherer) Loop(twitchUsername string) {
	if clientId == "" {
		return
	}

	twitchClient := twitch.NewClient(clientId)

	var channelId int = 0
	switch twitchUsername {
	case "lekkerspelen":
		channelId = 52385053
	case "serpentgameplay":
		channelId = 49901658
	}

	if channelId == 0 {
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-time.After(120 * time.Second):
		}

		s, err := twitchClient.GetCurrentStream(channelId)
		if err != nil {
			fmt.Printf("error getting stream info: %s", err)
		}

		c.mu.Lock()
		if c.ctx.Err() != nil {
			return
		}

		c.info.Datapoints = append(c.info.Datapoints, StreamInfoDatapoint{
			Title:     s.Channel.Status,
			Viewcount: s.Viewers,
			Game:      s.Game,
		})
		c.mu.Unlock()
	}
}
func (c *DatapointGatherer) Done() StreamInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancel()
	return c.info
}

func handleStream(ctx context.Context, chatClient *chat.Client, url string) {
	unlock := lm.Lock(url)
	defer unlock()

	twitchUsername := strings.TrimPrefix(url, "https://www.twitch.tv/")
	if twitchUsername == url {
		// stream url is not a twitch url
		twitchUsername = ""
	}

	hasChat := chatClient != nil && twitchUsername != ""

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

		log.Printf("starting download for %s (twitchUsername = %s, hasChat = %t)\n", url, twitchUsername, hasChat)

		gatherer := NewDatapointGatherer()
		go gatherer.Loop(twitchUsername)

		var f *os.File
		if hasChat {
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
		queue <- convert.Item{outputFile, 0}

		if f != nil {
			chatClient.RemoveChatFunction(twitchUsername)
			f.Close()
		}

		// write yaml information about stream to file
		streamInfo := gatherer.Done()
		if err := writeYamlFile(outputFile, &streamInfo); err != nil {
			log.Printf("error while writing yaml file for %s: %s\n", url, err)
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

	var chatClient *chat.Client
	if val := os.Getenv("TWITCH_AUTH"); val != "" {
		splitted := strings.SplitN(val, ":", 2)
		username := splitted[0]
		apiKey := splitted[1]

		log.Printf("created chat client for username %s", username)

		chatClient = chat.CreateClient(username, apiKey)
		go func() {
			err := chatClient.Connect()
			if err != nil {
				log.Printf("error connecting to twitch irc: %s", err)
			}
		}()
	}

	clientId = os.Getenv("TWITCH_CLIENT_ID")

	files, err := readDirRecursive(mainDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if filepath.Ext(f) != ".ts" {
			continue
		}

		queue <- convert.Item{f, 0}
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
