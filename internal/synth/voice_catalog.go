package synth

import (
	"fmt"
	"strconv"
	"strings"
)

func KnownVoicesForModelPath(modelPath string) []string {
	_ = modelPath
	return nil
}

func ResolveSpeakerID(modelPath, voice string) (int, error) {
	_ = modelPath
	v := strings.TrimSpace(strings.ToLower(voice))
	if v == "" {
		return 0, nil
	}
	if sid, err := strconv.Atoi(v); err == nil {
		if sid < 0 {
			return 0, fmt.Errorf("voice sid must be >= 0")
		}
		return sid, nil
	}
	return 0, fmt.Errorf("voice names are not hardcoded; use numeric sid")
}
