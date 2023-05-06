package manifest

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/iloahz/netrics/logs"
	"github.com/iloahz/netrics/utils"
	"go.uber.org/zap"
	"moul.io/http2curl"
)

type Resource struct {
	URL  string `json:"url"`
	CURL string `json:"curl"`
}

type Website struct {
	URL       string     `json:"url"`
	Resources []Resource `json:"resources"`
	Updated   string     `json:"updated"`
}

func requestToCurl(req *network.Request) (string, error) {
	httpReq, err := http.NewRequest(req.Method, req.URL, strings.NewReader(req.PostData))
	if err != nil {
		return "", err
	}
	httpReq.Header = make(http.Header)
	for k, v := range req.Headers {
		sv, ok := v.(string)
		if !ok {
			continue
		}
		httpReq.Header.Add(k, sv)
	}
	cmd, err := http2curl.GetCurlCommand(httpReq)
	if err != nil {
		return "", err
	}
	return cmd.String(), nil
}

func SummarizeWebsite(url string) (*Website, error) {
	logs.Info("summarizing website", zap.String("url", url))
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	timer := utils.NewTimerWithContext(time.Second, ctx)
	resources := make(chan Resource, 999)
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			logs.Debug("new request", zap.String("url", e.Request.URL))
			timer.Stop()
			curl, err := requestToCurl(e.Request)
			if err != nil {
				return
			}
			resources <- Resource{
				URL:  e.Request.URL,
				CURL: curl,
			}
		case *page.EventLifecycleEvent:
			if e.Name == "networkIdle" {
				logs.Debug("networkIdle")
				timer.Start()
			}
		default:
		}
	})
	err := chromedp.Run(
		ctx,
		chromedp.Navigate(url),
	)
	if err != nil {
		return nil, err
	}
	<-timer.Done
	logs.Debug("waited 1 second after network idle, no new request, consider page fully loaded")
	res := &Website{
		URL:       url,
		Resources: []Resource{},
		Updated:   time.Now().UTC().Format(time.RFC3339),
	}
	close(resources)
	for r := range resources {
		res.Resources = append(res.Resources, r)
	}
	return res, nil
}
