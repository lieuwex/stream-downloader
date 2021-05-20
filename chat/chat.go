package chat

import (
	"fmt"
	"sync"
	"time"

	twitch "github.com/gempir/go-twitch-irc/v2"
)

type Message struct {
	Type    string            `json:"type"`
	Tags    map[string]string `json:"tags"`
	Time    time.Time         `json:"time"`
	Message string            `json:"message"`
}

type ChatCallback func(Message)

type Client struct {
	ircClient *twitch.Client

	mu    sync.Mutex
	fnmap map[string]ChatCallback
}

func CreateClient() *Client {
	client := &Client{
		ircClient: twitch.NewAnonymousClient(),

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

	client.ircClient.OnPrivateMessage(func(m twitch.PrivateMessage) { fn("chat", m.Channel, m.Message, m.Tags) })
	client.ircClient.OnUserNoticeMessage(func(m twitch.UserNoticeMessage) { fn(m.Tags["msg-id"], m.Channel, m.Message, m.Tags) })
	//client.ircClient.OnWhisperMessage(func(m twitch.WhisperMessage) { })
	//client.ircClient.OnClearChatMessage(func(m twitch.ClearChatMessage) {})
	//client.ircClient.OnClearMessage(func(m twitch.ClearMessage) {})
	//client.ircClient.OnRoomStateMessage(func(m twitch.RoomStateMessage) {})
	//client.ircClient.OnUserStateMessage(func(m twitch.UserStateMessage) {})
	//client.ircClient.OnGlobalUserStateMessage(func(m twitch.GlobalUserStateMessage) {})
	//client.ircClient.OnNoticeMessage(func(m twitch.NoticeMessage) {})
	//client.ircClient.OnUserJoinMessage(func(m twitch.UserJoinMessage) {})
	//client.ircClient.OnUserPartMessage(func(m twitch.UserPartMessage) {})

	return client
}

func (c *Client) Connect() error {
	return c.ircClient.Connect()
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
	c.ircClient.Depart(channel)
}
