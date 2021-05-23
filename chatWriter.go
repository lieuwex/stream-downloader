package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"stream-downloader/chat"
	"time"

	"github.com/klauspost/compress/zstd"
)

func formatChatMessage(msg chat.Message, time time.Time) ([]byte, error) {
	var buf bytes.Buffer

	bytes, err := time.MarshalText()
	if err != nil {
		return []byte{}, err
	}
	buf.Write(bytes)
	buf.WriteByte(' ')

	enc := json.NewEncoder(&buf)
	if err := enc.Encode(msg); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// takes ownership of f, borrows chatClient
func chatRoutine(ctx context.Context, f *os.File, channel string, chatClient *chat.Client) {
	defer f.Close()

	wr, err := zstd.NewWriter(f, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		panic(err)
	}
	defer wr.Close()

	chatClient.AddChatFunction(channel, func(msg chat.Message, time time.Time) {
		line, err := formatChatMessage(msg, time)
		if err != nil {
			log.Printf("error while marshalling message: %s", err)
		}

		if _, err := wr.Write(line); err != nil {
			log.Printf("error while writing chat message to file: %s", err)
		}
	})
	defer chatClient.RemoveChatFunction(channel)

	<-ctx.Done()
}
