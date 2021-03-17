package chat

import (
	"fmt"
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
	username  string
	apiKey    string
	ircClient *gotirc.Client

	mu    sync.Mutex
	fnmap map[string]ChatCallback
}

func CreateClient(username, apiKey string) *Client {
	options := gotirc.Options{
		Host: "irc.chat.twitch.tv",
		Port: 6667,
	}

	client := &Client{
		username:  username,
		apiKey:    apiKey,
		ircClient: gotirc.NewClient(options),

		fnmap: make(map[string]ChatCallback),
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

func (c *Client) Connect() error {
	return c.ircClient.Connect(c.username, c.apiKey)
}

func (c *Client) rejoinChats() {
	c.mu.Lock()
	defer c.mu.Lock()

	c.ircClient.Disconnect()
	c.Connect()
	for key, _ := range c.fnmap {
		c.ircClient.Join(key)
	}
}

func (c *Client) AddChatFunction(channel string, cb ChatCallback) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("recovered in AddChatFunction,", e)
			fmt.Println("rejoining chats...")
			c.rejoinChats()
		}
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.fnmap[channel] = cb
	c.ircClient.Join(channel)
}

func (c *Client) RemoveChatFunction(channel string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("recovered in RemoveChatFunction", e)
			fmt.Println("rejoining chats...")
			c.rejoinChats()
		}
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.fnmap, channel)
	c.ircClient.Part(channel)
}
