package manifest

import (
	"encoding/json"
	"os"
	"time"

	"github.com/iloahz/netrics/logs"
	"go.uber.org/zap"
)

type Manifest struct {
	Websites map[string]Website `json:"websites"`
}

func UpdateManifest(file string) error {
	buf, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var manifest Manifest
	err = json.Unmarshal(buf, &manifest)
	if err != nil {
		return err
	}
	updatedManifest := Manifest{
		Websites: map[string]Website{},
	}
	for title, website := range manifest.Websites {
		updatedManifest.Websites[title] = website
	}
	for title, website := range manifest.Websites {
		lastUpdated, err := time.Parse(time.RFC3339, website.Updated)
		if err != nil {
			lastUpdated = time.UnixMilli(0)
		}
		if time.Since(lastUpdated) < time.Hour {
			logs.Info("updated recently, skip", zap.String("website", title))
			continue
		}
		updated, err := SummarizeWebsite(website.URL)
		if err != nil {
			return err
		}
		updatedManifest.Websites[title] = *updated
		buf, err = json.MarshalIndent(updatedManifest, "", "    ")
		if err != nil {
			return err
		}
		err = os.WriteFile(file, buf, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
