package twitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	clientId   string
}

func NewClient(clientId string) Client {
	return Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		clientId:   clientId,
	}
}

func (c Client) getUrl(channelId int) string {
	return fmt.Sprintf("https://api.twitch.tv/kraken/streams/%d?client_id=%s", channelId, c.clientId)
}

type streamResponse struct {
	Stream *Stream `json:"stream"`
}

func (c Client) GetCurrentStream(channelId int) (*Stream, error) {
	req, err := http.NewRequest("GET", c.getUrl(channelId), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var resObj streamResponse
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&resObj); err != nil {
		return nil, err
	}

	return resObj.Stream, nil
}
