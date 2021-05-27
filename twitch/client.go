package twitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const httpClientTimeout = 5 * time.Second

type Client struct {
	httpClient *http.Client
	clientId   string
}

func NewClient(clientId string) Client {
	return Client{
		httpClient: &http.Client{Timeout: httpClientTimeout},
		clientId:   clientId,
	}
}

func (c Client) getUrl(slug string) string {
	sep := '?'
	if strings.ContainsRune(slug, '?') {
		sep = '&'
	}
	return fmt.Sprintf("https://api.twitch.tv/kraken%s%cclient_id=%s", slug, sep, c.clientId)
}

func (c Client) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")

	return c.httpClient.Do(req)
}

func (c Client) GetCurrentStream(channelId string) (*Stream, error) {
	url := c.getUrl(fmt.Sprintf("/streams/%s", channelId))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var resObj struct {
		Stream *Stream `json:"stream"`
	}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&resObj); err != nil {
		return nil, err
	}

	return resObj.Stream, nil
}

func (c Client) GetChannelId(channelName string) (string, error) {
	url := c.getUrl(fmt.Sprintf("/users?login=%s", channelName))
	println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	res, err := c.doRequest(req)
	if err != nil {
		return "", err
	}

	var resObj struct {
		Users []struct {
			Id string `json:"_id"`
		} `json:"users"`
	}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&resObj); err != nil {
		return "", err
	}

	if len(resObj.Users) == 0 {
		return "", fmt.Errorf("no user found wth name %s", channelName)
	}
	return resObj.Users[0].Id, nil
}
