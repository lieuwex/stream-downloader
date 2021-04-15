package main

type StreamInfoDatapoint struct {
	Title     string
	Viewcount int64
	Game      string
}

type StreamInfo struct {
	Datapoints []StreamInfoDatapoint
	Jumpcuts   []interface{} // TODO (and REVIEW?), although stream-downloader shouldn't really care about this
}
