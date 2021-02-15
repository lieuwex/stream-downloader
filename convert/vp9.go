package convert

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type VP9Settings struct {
	CRF int

	MinBirate     int
	TargetBitrate int
	MaxBitrate    int
}

var Default = &VP9Settings{
	CRF:           32,
	MinBirate:     0,
	TargetBitrate: 2800,
	MaxBitrate:    4350,
}

func GetSettings(width, height, fps int) *VP9Settings {
	if width == 1280 && height == 720 && fps == 30 {
		// 720p30
		log.Println("using settings for 720p30")
		return &VP9Settings{
			CRF:           33,
			MinBirate:     0,
			TargetBitrate: 1800,
			MaxBitrate:    2610,
		}
	} else if width == 1280 && height == 720 && fps == 60 {
		// 720p60
		log.Println("using settings for 720p60")
		return &VP9Settings{
			CRF:           33,
			MinBirate:     0,
			TargetBitrate: 1800,
			MaxBitrate:    2610,
		}
	} else if width == 1920 && height == 1080 && fps == 30 {
		// 1080p30
		log.Println("using settings for 1080p30")
		return &VP9Settings{
			CRF:           32,
			MinBirate:     0,
			TargetBitrate: 1800,
			MaxBitrate:    2610,
		}
	} else {
		// 1080p60, and a fallback. This is fairly high.
		log.Println("using settings for 1080p60")
		return Default
	}
}

func GetSettingsByFile(path string) (*VP9Settings, error) {
	b, err := exec.Command(
		"ffprobe",
		"-select_streams",
		"v:0",
		"-show_entries",
		"stream=width,height,avg_frame_rate",
		"-of",
		"csv=p=0",
		path,
	).Output()
	if err != nil {
		return nil, err
	}

	items := strings.Split(string(b), ",")
	width, err := strconv.Atoi(items[0])
	if err != nil {
		return nil, err
	}

	height, err := strconv.Atoi(items[1])
	if err != nil {
		return nil, err
	}

	fpsRaw := strings.TrimSuffix(items[2], "/1")
	fps, err := strconv.Atoi(fpsRaw)
	if err != nil {
		return nil, err
	}

	return GetSettings(width, height, fps), nil
}
