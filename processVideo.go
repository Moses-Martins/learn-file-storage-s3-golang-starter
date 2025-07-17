package main

import (
    "bytes"
    "fmt"
    "os/exec"
)


func processVideoForFastStart(filePath string) (string, error) {
	output := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", output)
	
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	
	if err := cmd.Run(); err != nil {
		 return "", fmt.Errorf("failed to run ffmpeg: %w\nOutput: %s", err, out.String())
	}

	return output, nil

}