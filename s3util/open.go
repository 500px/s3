package s3util

import (
	"io"
	"net/http"
	"time"
)

type metricsReadCloserDecorator struct {
	body            io.ReadCloser
	metricsCallback MetricsCallbackFunc
	totalTime       time.Duration
}

// Open requests the S3 object at url. An HTTP status other than 200 is
// considered an error.
//
// If c is nil, Open uses DefaultConfig.
func Open(url string, c *Config) (io.ReadCloser, error) {
	if c == nil {
		c = DefaultConfig
	}
	// TODO(kr): maybe parallel range fetching
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	c.Sign(r, *c.Keys)
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	start := time.Now()
	resp, err := client.Do(r)
	end := time.Now()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, newRespError(resp)
	}
	return &metricsReadCloserDecorator{
		body:            resp.Body,
		metricsCallback: c.MetricsCallback,
		totalTime:       end.Sub(start),
	}, nil
}

func (m *metricsReadCloserDecorator) Read(p []byte) (n int, err error) {
	n, err = m.body.Read(p)

	if m.metricsCallback != nil {
		m.metricsCallback(
			Metrics{
				TotalBytes: uint64(n),
				TotalTime:  m.totalTime,
			})
	}

	return n, err
}

func (m *metricsReadCloserDecorator) Close() error {
	return m.body.Close()
}
