package main


import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "math"
    "os/exec"
)

type FFProbeOutput struct {
    Streams []struct {
        Width  int `json:"width"`
        Height int `json:"height"`
    } `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	
	var out bytes.Buffer
	cmd.Stdout = &out
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run ffprobe: %w", err)
	}


	probe := FFProbeOutput{}
	if err := json.Unmarshal(out.Bytes(), &probe); err != nil {
        return "", fmt.Errorf("failed to parse ffprobe output: %w", err)
    }


	if len(probe.Streams) == 0 || probe.Streams[0].Width == 0 || probe.Streams[0].Height == 0 {
        return "", errors.New("no valid video stream found")
    }

	width := probe.Streams[0].Width
    height := probe.Streams[0].Height

    ratio := float64(width) / float64(height)

    // Determine ratio string
    const epsilon = 0.05 // tolerance for floating point comparison
    switch {
    case math.Abs(ratio-16.0/9.0) < epsilon:
        return "16:9", nil
    case math.Abs(ratio-9.0/16.0) < epsilon:
        return "9:16", nil
    default:
        return "other", nil
    }


}






