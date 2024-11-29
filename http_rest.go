package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type RequestError struct {
	StatusCode int
	Body       string
	Err        error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("status %d: %s", e.StatusCode, e.Body)
}

type HttpClient struct {
	client      *http.Client
	baseDelay   time.Duration
	maxAttempts int
	logger      *zap.SugaredLogger
}

func NewHttpClient(client *http.Client, logger *zap.SugaredLogger) *HttpClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HttpClient{
		client:      client,
		baseDelay:   time.Second,
		maxAttempts: 3,
		logger:      logger,
	}
}

func calcBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// use bit shfiting for int exponential growth: 2^n
	backoff := baseDelay * time.Duration(1<<time.Duration(attempt))

	// add +/- 20% jitter
	jitter := time.Duration(rand.Float64()*0.4-0.2) * backoff
	backoff += jitter

	maxBackoff := 30 * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

func isRetryableStatusCode(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout ||
		code >= 500
}

func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

func (c *HttpClient) execReq(req *http.Request, attempts int) ([]byte, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		start := time.Now()

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if i < attempts-1 {
				c.logger.Warnw("retrying failed request",
					"attempt", i+1,
					"error", err,
					"url", req.URL.String())
				time.Sleep(calcBackoff(i, c.baseDelay))
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		duration := time.Since(start)

		c.logger.Debugw("request completed",
			"method", req.Method,
			"url", req.URL.String(),
			"status", resp.StatusCode,
			"duration", duration)

		if err != nil {
			lastErr = fmt.Errorf("reading response: %w", err)
			if i < attempts-1 {
				time.Sleep(calcBackoff(i, c.baseDelay))
				continue
			}
			return nil, lastErr
		}

		if !isSuccessStatus(resp.StatusCode) {
			lastErr = &RequestError{
				StatusCode: resp.StatusCode,
				Body:       string(body),
			}
			if i < attempts-1 && isRetryableStatusCode(resp.StatusCode) {
				time.Sleep(calcBackoff(i, c.baseDelay))
				continue
			}
			return nil, lastErr
		}

		return body, nil
	}
	return nil, fmt.Errorf("request failed after %d attempts: %v", attempts, lastErr)
}

func (c *HttpClient) PostJsonReq(ctx context.Context, url string, payload interface{}, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.execReq(req, c.maxAttempts)
}

func (c *HttpClient) PostFormReq(ctx context.Context, url string, formData url.Values, headers map[string]string) ([]byte, error) {
	encodedData := formData.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(encodedData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(encodedData)))

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.execReq(req, c.maxAttempts)
}

func (c *HttpClient) GetReq(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.execReq(req, c.maxAttempts)
}

func (c *HttpClient) PutJsonReq(ctx context.Context, url string, payload interface{}, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.execReq(req, c.maxAttempts)
}

func (c *HttpClient) PatchJsonReq(ctx context.Context, url string, payload interface{}, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.execReq(req, c.maxAttempts)
}

func (c *HttpClient) DeleteReq(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.execReq(req, c.maxAttempts)
}
