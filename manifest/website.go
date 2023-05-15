package manifest

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/iloahz/netrics/logs"
	"github.com/iloahz/netrics/utils"
	"go.uber.org/zap"
)

type Resource struct {
	URL          string `json:"url"`
	ContentType  string `json:"content_type"`
	Order        int    `json:"order"`
	Dependencies []int  `json:"dependencies"`
}

type Website struct {
	URL       string     `json:"url"`
	Favicon   string     `json:"favicon"`
	Resources []Resource `json:"resources"`
	Updated   string     `json:"updated"`
}

func shouldKeep(e *network.EventRequestWillBeSent) bool {
	if !strings.EqualFold(e.Request.Method, "GET") {
		return false
	}
	if strings.HasPrefix(e.Request.URL, "data:") {
		return false
	}
	if e.Type == network.ResourceTypeImage {
		u, err := url.Parse(e.Request.URL)
		if err != nil {
			return false
		}
		if len(u.Query()) > 3 || len(u.RawQuery) > 100 {
			// likely a tracking gif
			return false
		}
	}
	return true
}

func SummarizeWebsite(url string) (*Website, error) {
	logs.Info("summarizing website", zap.String("url", url))
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	timer := utils.NewTimerWithContext(time.Second*3, ctx)
	var mutex sync.Mutex
	type RequestInfo struct {
		ID           string
		URL          string
		Type         string
		Order        int
		Dependencies []int
		ShouldKeep   bool
	}
	idToInfo := map[string]*RequestInfo{}
	finishedList := []int{}
	var order int
	var sent int
	var finished int
	var failed int
	withNetworkStats := func(fields ...zap.Field) []zap.Field {
		return append(fields,
			zap.Int("sent", sent),
			zap.Int("finished", finished),
			zap.Int("failed", failed),
			zap.Int("pending", sent-finished-failed),
		)
	}
	getInfo := func(id string) *RequestInfo {
		info, ok := idToInfo[id]
		if !ok {
			info = &RequestInfo{}
			idToInfo[id] = info
		}
		return info
	}
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		mutex.Lock()
		defer mutex.Unlock()
		// case *network.EventResponseReceivedExtraInfo:
		// case *network.EventDataReceived:
		// case *network.EventLoadingFinished:
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			info := getInfo(string(e.RequestID))
			if len(info.ID) == 0 {
				sent += 1
				order += 1
				info.ID = string(e.RequestID)
				info.URL = e.Request.URL
				info.Type = e.Type.String()
				info.Order = order
				info.Dependencies = finishedList[:]
				info.ShouldKeep = shouldKeep(e)
			}
			logs.Debug("new request", withNetworkStats(zap.String("url", e.Request.URL), zap.String("id", e.RequestID.String()))...)
			timer.Stop()
		case *network.EventResponseReceived:
			finished += 1
			info := getInfo(string(e.RequestID))
			if info.ShouldKeep {
				finishedList = append(finishedList, info.Order)
			}
			logs.Debug("finished request", withNetworkStats(zap.String("id", e.RequestID.String()), zap.Float64("size", e.Response.EncodedDataLength))...)
			if sent-finished-failed <= 0 {
				logs.Debug("0 pending request, counting down")
				timer.Start()
			}
		case *network.EventLoadingFailed:
			failed += 1
			logs.Warn("failed request", withNetworkStats(zap.String("id", e.RequestID.String()))...)
			if sent-finished-failed <= 0 {
				logs.Debug("0 pending request, counting down")
				timer.Start()
			}
		case *page.EventLifecycleEvent:
			if e.Name == "networkIdle" {
				logs.Debug("networkIdle, counting down")
				timer.Start()
			}
		default:
		}
	})
	js := `
	// query the first <link> element whose rel attribute contains the word icon 
	const iconElement = document.querySelector("link[rel~=icon]");
	// fallback to "/favicon.ico" in case it's not specified
	const href = (iconElement && iconElement.href) || "/favicon.ico";
	// make sure it's an absolute URL
	const faviconURL = new URL(href, window.location).toString();
	// the value to return
	faviconURL`
	var faviconURL string
	err := chromedp.Run(
		ctx,
		chromedp.Navigate(url),
		chromedp.ActionFunc(func(ctx context.Context) error {
			<-timer.Done
			return nil
		}),
		chromedp.Evaluate(js, &faviconURL),
	)
	if err != nil {
		return nil, err
	}
	logs.Debug("waited 3 seconds after network idle, no new request, consider page fully loaded")
	res := &Website{
		URL:       url,
		Favicon:   faviconURL,
		Resources: []Resource{},
		Updated:   time.Now().UTC().Format(time.RFC3339),
	}
	for _, info := range idToInfo {
		if info.ShouldKeep {
			res.Resources = append(res.Resources, Resource{
				URL:          info.URL,
				ContentType:  info.Type,
				Order:        info.Order,
				Dependencies: info.Dependencies,
			})
		}
	}
	sort.Slice(res.Resources, func(i, j int) bool {
		return res.Resources[i].Order < res.Resources[j].Order
	})
	return res, nil
}
