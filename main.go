package main

import (
	"context"
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

func twitchInfoLoop(ctx context.Context, twitchUsername, outputFile string) {
	if clientId == "" {
		fmt.Println("clientId == \"\"")
		return
	}

	twitchClient := twitch.NewClient(clientId)

	channelId, err := twitchClient.GetChannelId(twitchUsername)
	if err != nil {
		log.Printf("error getting stream id: %s. Falling back.", err)

		switch twitchUsername {
		case "lekkerspelen":
			channelId = "52385053"
		case "serpentgameplay":
			channelId = "49901658"

		default:
			log.Printf("no fallback possible")
			channelId = ""
		}
	}
	fmt.Printf("channelId = %s\n", channelId)
	if channelId == "" {
		return
	}

	handleTick := func(info *StreamInfo) error {
		s, err := twitchClient.GetCurrentStream(channelId)
		if err != nil {
			return fmt.Errorf("error getting stream info: %s", err)
		} else if s == nil {
			return fmt.Errorf("got empty datapoint from twitch")
		}

		info.Datapoints = append(info.Datapoints, StreamInfoDatapoint{
			Title:     s.Channel.Status,
			Viewcount: s.Viewers,
			Game:      s.Game,
			Timestamp: time.Now().Unix(),
		})

		return writeYamlFile(outputFile, info)
	}

	var info StreamInfo

	for {
		if err := handleTick(&info); err != nil {
			log.Printf("error while ticking twitch info for %s: %s\n", twitchUsername, err)
		}

		select {
		case <-ctx.Done():
			fmt.Println("context has been finished while waiting, goodbye")
			return

		case <-time.After(120 * time.Second):
		}
	}
}

func handleStream(channelCtx context.Context, chatClient *chat.Client, url string) {
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
		case <-channelCtx.Done():
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

		streamCtx, cancelStreamCtx := context.WithCancel(context.Background())

		go twitchInfoLoop(streamCtx, twitchUsername, outputFile)

		if hasChat {
			f, err := os.Create(strings.Replace(outputFile, ".ts", ".txt.zst", 1))
			if err != nil {
				log.Printf("error while creating chat output file: %s", err)
			} else {
				go chatRoutine(streamCtx, f, twitchUsername, chatClient)
			}
		}

		cmd := streamlink.GetDownloadCommand(url, outputFile)
		if err := cmd.Run(); err != nil {
			log.Printf("error while running streamlink for %s: %s\n", url, err)
		}

		log.Printf("stream for %s ended\n", url)
		cancelStreamCtx()
		queue <- convert.Item{outputFile, 0}
	}
}

func parseStreamList(path string) ([]string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, err
	}

	var res []string
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		line := strings.TrimSpace(line)
		if line == "" {
			continue
		}
		res = append(res, line)
	}
	return res, nil
}

func main() {
	mainDir = os.Args[1]

	chatClient := chat.CreateClient()
	log.Println("created chat client")
	go func() {
		err := chatClient.Connect()
		if err != nil {
			log.Printf("error connecting to twitch irc: %s", err)
		}
	}()

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
