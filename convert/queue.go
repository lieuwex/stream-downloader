package convert

import (
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const maxTryCount = 5

type Settings struct {
	Size int

	ThreadCount int

	VideoWidth  int
	VideoHeight int
}

func convertStreamFile(settings Settings, input string) error {
	dir, inputFile := filepath.Split(input)

	timestamp := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	output := filepath.Join(dir, timestamp+".webm")

	log.Printf("starting converting %s to %s", input, output)

	vp9Settings, err := GetSttingsByFile(input)
	if err != nil {
		vp9Settings = Default
	}

	if err := exec.Command(
		"/bin/convert",

		input,
		strconv.Itoa(settings.ThreadCount),
		strconv.Itoa(vp9Settings.CRF),
		strconv.Itoa(vp9Settings.MinBirate),
		strconv.Itoa(vp9Settings.TargetBitrate),
		strconv.Itoa(vp9Settings.MaxBitrate),
		strconv.Itoa(settings.VideoWidth),
		strconv.Itoa(settings.VideoHeight),
		output,
	).Run(); err != nil {
		return err
	}

	log.Printf("done converting %s to %s", input, output)

	// TODO: touch -r

	log.Printf("keeping original file %s", input)
	//if err := os.Remove(input); err != nil {
	//	log.Printf("error while removing input file %s: %s", input, err)
	//}

	return nil
}

type Item struct {
	Path     string
	TryCount uint
}
type Queue chan Item

func MakeQueue(settings Settings) Queue {
	ch := make(chan Item, settings.Size)

	go func() {
		for item := range ch {
			if item.TryCount == maxTryCount {
				log.Printf("too much tries for %s", item.Path)
				continue
			}

			if err := convertStreamFile(settings, item.Path); err != nil {
				log.Printf("error while converting %s: %s, trying again", item.Path, err)
				ch <- Item{item.Path, item.TryCount + 1}
			}
		}
	}()

	return ch
}
