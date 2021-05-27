package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"stream-downloader/twitch"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

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
		log.Printf("error getting stream id for %s: %s. Falling back.", twitchUsername, err)

		switch twitchUsername {
		case "lekkerspelen":
			channelId = "52385053"
		case "serpentgameplay":
			channelId = "49901658"

		default:
			log.Printf("no fallback found for %s.", twitchUsername)
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

		case <-time.After(twitchLoopInterval):
		}
	}
}
