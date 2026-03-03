// Package playback provides local speaker playback helpers.
package playback

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func PlayFile(path, format string, sampleRate int) error {
	if _, err := exec.LookPath("aplay"); err != nil {
		return fmt.Errorf("aplay not found in PATH")
	}
	var cmd *exec.Cmd
	if format == "pcm_s16le" {
		rate := sampleRate
		if rate <= 0 {
			rate = 22050
		}
		cmd = exec.Command("aplay", "-q", "-f", "S16_LE", "-c", "1", "-r", fmt.Sprintf("%d", rate), path)
	} else {
		cmd = exec.Command("aplay", "-q", path)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func PlayBytes(data []byte, format string, sampleRate int) error {
	if _, err := exec.LookPath("aplay"); err != nil {
		return fmt.Errorf("aplay not found in PATH")
	}
	var cmd *exec.Cmd
	if format == "pcm_s16le" {
		rate := sampleRate
		if rate <= 0 {
			rate = 22050
		}
		cmd = exec.Command("aplay", "-q", "-f", "S16_LE", "-c", "1", "-r", fmt.Sprintf("%d", rate), "-")
	} else {
		cmd = exec.Command("aplay", "-q", "-")
	}
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
