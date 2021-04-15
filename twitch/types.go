package twitch

type Channel struct {
	Status string `json:"status"`
}

type Stream struct {
	Viewers    int64   `json:"viewers"`
	Game       string  `json:"game"`
	StreamType string  `json:"stream_type"`
	Channel    Channel `json:"channel"`
}
