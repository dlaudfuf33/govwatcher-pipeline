package util

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gwatch-data-pipeline/internal/logging"
)

func GetLatestNoticeFilePath() string {
	noticeDir := "./downloads/notice"
	var latestPath string
	var latestTime time.Time

	err := filepath.Walk(noticeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "legislation_notice_") && strings.HasSuffix(name, ".xlsx") {
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestPath = path
			}
		}
		return nil
	})
	if err != nil {
		logging.Errorf("Failed to walk notice dir: %v", err)
		return ""
	}
	return latestPath
}
func GetNA() string{
	apiKey := os.Getenv("NA_KEY")
	if apiKey == "" {
		logging.Errorf("Missing API Key API key is missing in environment (NA_KEY).")
		return ""
	}else{
		return apiKey
	}
}

func MakeRequestWithUA(method string, url string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed create request : %v", err)
	}

	req.Header.Add("User-Agent", "GWatchBot/1.0 (+https://gwatch.example.com)")

	return client.Do(req)
}
