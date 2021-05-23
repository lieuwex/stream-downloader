package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"stream-downloader/chat"
	"strings"
	"time"
)

func formatChatMessage(msg chat.Message, time time.Time) (string, error) {
	var buf strings.Builder

	bytes, err := time.MarshalText()
	if err != nil {
		return "", err
	}
	buf.Write(bytes)
	buf.WriteByte(' ')

	enc := json.NewEncoder(&buf)
	if err := enc.Encode(msg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// takes ownership of f, borrows chatClient
func chatRoutine(ctx context.Context, f *os.File, twitchUsername string, chatClient *chat.Client) {
	chatClient.AddChatFunction(twitchUsername, func(msg chat.Message, time time.Time) {
		str, err := formatChatMessage(msg, time)
		if err != nil {
			log.Printf("error while marshalling message: %s", err)
		}
		if _, err := f.WriteString(str); err != nil {
			log.Printf("error while writing chat message to file: %s", err)
		}
	})

	<-ctx.Done()

	chatClient.RemoveChatFunction(twitchUsername)
	f.Close()
}
