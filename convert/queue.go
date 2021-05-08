package convert

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// retry a stream up to 5 times
	maxTryCount = 5
	// handle up to 2 streams at once
	workerCount = 2
)

type Settings struct {
	Size int

	MaxVideoWidth  int
	MaxVideoHeight int
}

func convertStreamFile(settings Settings, input string) error {
	dir, inputFile := filepath.Split(input)

	timestamp := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
	output := filepath.Join(dir, timestamp+".webm")

	log.Printf("starting converting %s to %s", input, output)

	vp9Settings, err := GetSettingsByFile(input)
	if err != nil {
		log.Printf("error getting vp9 settings: %s. Falling back on default settings.", err)
		vp9Settings = Default
	}

	if err := exec.Command(
		"/bin/convert",

		input,
		strconv.Itoa(vp9Settings.CRF),
		strconv.Itoa(vp9Settings.MinBirate),
		strconv.Itoa(vp9Settings.TargetBitrate),
		strconv.Itoa(vp9Settings.MaxBitrate),
		strconv.Itoa(settings.MaxVideoWidth),
		strconv.Itoa(settings.MaxVideoHeight),
		output,
	).Run(); err != nil {
		return err
	}

	log.Printf("done converting %s to %s", input, output)

	// TODO: touch -r

	inputNewPath := input + ".bak"
	log.Printf("renaming original file %s -> %s", input, inputNewPath)
	if err := os.Rename(input, inputNewPath); err != nil {
		log.Printf("error while renaming input file %s: %s", input, err)
	}

	return nil
}

type Item struct {
	Path     string
	TryCount uint
}
type Queue chan Item

func MakeQueue(settings Settings) Queue {
	ch := make(chan Item, settings.Size)

	for i := 0; i < workerCount; i++ {
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
	}

	return ch
}
