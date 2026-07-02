package urlcheck

import (
	"io"
	"net"
	"net/http"
	"time"
)

type Checker interface {
	Check(url string) (int, error)
}

type HTTPChecker struct {
	client http.Client
}

func NewChecker() HTTPChecker {
	transport := &http.Transport{
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		DialContext: (&net.Dialer{
			Timeout: 2 * time.Second,
		}).DialContext,
	}
	return HTTPChecker{
		client: http.Client{
			Timeout:   3 * time.Second,
			Transport: transport,
		},
	}
}

func (checker HTTPChecker) Check(url string) (int, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := checker.client.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, nil
}
