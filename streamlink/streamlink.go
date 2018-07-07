package streamlink

import (
	"os/exec"
	"syscall"
)

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

func GetDownloadCommand(url, outputFile string) *exec.Cmd {
	return exec.Command(
		"streamlink",
		"--twitch-disable-hosting",
		url,
		"1080p,720p,best",
		"-o",
		outputFile,
	)
}
