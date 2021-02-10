package chat

import (
	"sync"
	"time"

	"github.com/jrm780/gotirc"
)

type Message struct {
	Type    string            `json:"type"`
	Tags    map[string]string `json:"tags"`
	Time    time.Time         `json:"time"`
	Message string            `json:"message"`
}

type ChatCallback func(Message)

type Client struct {
	ircClient *gotirc.Client

	mu    sync.Mutex
	fnmap map[string]ChatCallback
}

func CreateClient() *Client {
	options := gotirc.Options{
		Host: "irc.chat.twitch.tv",
		Port: 6667,
	}

	client := &Client{
		fnmap:     make(map[string]ChatCallback),
		ircClient: gotirc.NewClient(options),
	}

	fn := func(typ, channel, msg string, tags map[string]string) {
		client.mu.Lock()
		fn, has := client.fnmap[channel[1:]]
		client.mu.Unlock()
		if !has {
			return
		}

		fn(Message{
			Type:    typ,
			Tags:    tags,
			Time:    time.Now(),
			Message: msg,
		})
	}

	client.ircClient.OnAction(func(channel string, tags map[string]string, msg string) { fn("action", channel, msg, tags) })
	client.ircClient.OnChat(func(channel string, tags map[string]string, msg string) { fn("chat", channel, msg, tags) })
	client.ircClient.OnCheer(func(channel string, tags map[string]string, msg string) { fn("cheer", channel, msg, tags) })
	client.ircClient.OnJoin(func(channel string, msg string) { fn("join", channel, msg, make(map[string]string)) })
	client.ircClient.OnPart(func(channel string, msg string) { fn("part", channel, msg, make(map[string]string)) })
	client.ircClient.OnResub(func(channel string, tags map[string]string, msg string) { fn("resub", channel, msg, tags) })
	client.ircClient.OnSubscription(func(channel string, tags map[string]string, msg string) { fn("subscription", channel, msg, tags) })
	client.ircClient.OnSubGift(func(channel string, tags map[string]string, msg string) { fn("subgift", channel, msg, tags) })

	return client
}

func (c *Client) Connect(username, apiKey string) error {
	return c.ircClient.Connect(username, apiKey)
}

func (c *Client) AddChatFunction(channel string, cb ChatCallback) {
	c.mu.Lock()
	c.ircClient.Join(channel)
	c.fnmap[channel] = cb
	c.mu.Unlock()
}

func (c *Client) RemoveChatFunction(channel string) {
	c.mu.Lock()
	c.ircClient.Part(channel)
	delete(c.fnmap, channel)
	c.mu.Unlock()
}
