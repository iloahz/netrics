package manifest

import (
	"github.com/iloahz/netrics/logs"
	"go.uber.org/zap"
)

type WebsiteConfig struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Input []WebsiteConfig

type Manifest struct {
	Websites map[string]Website `json:"websites"`
}

func BuildManifest(input *Input) (*Manifest, error) {
	manifest := &Manifest{
		Websites: map[string]Website{},
	}
	for _, config := range *input {
		updated, err := SummarizeWebsite(config.URL)
		if err != nil {
			logs.Warn("failed to update", zap.String("url", config.URL))
			continue
		}
		manifest.Websites[config.Title] = *updated
	}
	return manifest, nil
}
