package streamlink

import (
	"os/exec"
	"syscall"
)

// IsOnline returns whether or not the stream at the given url is online.
func IsOnline(url string) (bool, error) {
	checkCmd := exec.Command(
		"streamlink",
		"--twitch-disable-hosting",
		url,
	)

	checkCmd.Start()
	state, err := checkCmd.Process.Wait()
	if err != nil {
		return false, err
	}

	waitStatus := state.Sys().(syscall.WaitStatus)
	return waitStatus.ExitStatus() == 0, nil
}

// GetDownloadCommand returns the exec.Cmd to download the stream at the given
// url and save it to outputFile.
func GetDownloadCommand(url, outputFile string) *exec.Cmd {
	return exec.Command(
		"streamlink",
		"--twitch-disable-hosting",
		"--twitch-disable-ads",
		url,
		"best",
		"-o",
		outputFile,
	)
}
