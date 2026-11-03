package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

// Request defines the payload submitted to POST /api/proxy.
type Request struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	TimeoutMs int               `json:"timeoutMs,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
}

// TimingMetrics records sub-millisecond network duration statistics.
type TimingMetrics struct {
	DNSLookupMs    float64 `json:"dnsLookupMs"`
	TCPConnectMs   float64 `json:"tcpConnectMs"`
	TLSHandshakeMs float64 `json:"tlsHandshakeMs"`
	TTFBMs         float64 `json:"ttfbMs"`
	TotalMs        float64 `json:"totalMs"`
	SizeBytes      int64   `json:"sizeBytes"`
}

// Response defines the standardized output returned by the proxy.
type Response struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Timing     TimingMetrics     `json:"timing"`
	Error      string            `json:"error,omitempty"`
}

// Client executes HTTP requests with low-level network tracing.
type Client struct {
	httpClient *http.Client
}

// NewClient initializes a proxy HTTP client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			// Individual request timeout is handled via request context
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
	}
}

// Execute dispatches the request and gathers telemetry metrics.
func (c *Client) Execute(ctx context.Context, req Request) (*Response, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, errors.New("request URL cannot be empty")
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid URL scheme (must be http or https): %s", req.URL)
	}

	// SSRF Protection: Block AWS/GCP/Azure internal metadata addresses
	host := parsedURL.Hostname()
	if isBlockedMetadataHost(host) {
		return nil, fmt.Errorf("access to internal cloud metadata service is prohibited: %s", host)
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	timeout := 30 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		dnsStart, dnsDone       time.Time
		connStart, connDone     time.Time
		tlsStart, tlsDone       time.Time
		reqStart                = time.Now()
		firstByteTime           time.Time
		dnsDuration             time.Duration
		tcpDuration             time.Duration
		tlsDuration             time.Duration
		ttfbDuration            time.Duration
	)

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsDone = time.Now()
			dnsDuration = dnsDone.Sub(dnsStart)
		},
		ConnectStart: func(network, addr string) {
			connStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			connDone = time.Now()
			tcpDuration = connDone.Sub(connStart)
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsDone = time.Now()
			tlsDuration = tlsDone.Sub(tlsStart)
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Now()
			ttfbDuration = firstByteTime.Sub(reqStart)
		},
	}

	tracedCtx := httptrace.WithClientTrace(reqCtx, trace)

	var bodyReader io.Reader
	if req.Body != "" && method != http.MethodGet && method != http.MethodHead {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(tracedCtx, method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply headers
	for k, v := range req.Headers {
		// Prevent setting host header improperly
		if strings.EqualFold(k, "Host") {
			httpReq.Host = v
		} else {
			httpReq.Header.Set(k, v)
		}
	}

	// Default User-Agent if not set
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "APIDock/1.0 (Developer Workstation)")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		totalDuration := time.Since(reqStart)
		return &Response{
			Status:     0,
			StatusText: "Connection Error",
			Headers:    make(map[string]string),
			Body:       "",
			Timing: TimingMetrics{
				DNSLookupMs:    durationToMs(dnsDuration),
				TCPConnectMs:   durationToMs(tcpDuration),
				TLSHandshakeMs: durationToMs(tlsDuration),
				TTFBMs:         durationToMs(ttfbDuration),
				TotalMs:        durationToMs(totalDuration),
				SizeBytes:      0,
			},
			Error: err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	totalDuration := time.Since(reqStart)
	if ttfbDuration == 0 {
		ttfbDuration = totalDuration
	}

	respHeaders := make(map[string]string)
	for k, values := range resp.Header {
		respHeaders[k] = strings.Join(values, ", ")
	}

	bodyStr := string(bodyBytes)
	if readErr != nil {
		bodyStr = fmt.Sprintf("Error reading response body: %v", readErr)
	}

	return &Response{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Headers:    respHeaders,
		Body:       bodyStr,
		Timing: TimingMetrics{
			DNSLookupMs:    durationToMs(dnsDuration),
			TCPConnectMs:   durationToMs(tcpDuration),
			TLSHandshakeMs: durationToMs(tlsDuration),
			TTFBMs:         durationToMs(ttfbDuration),
			TotalMs:        durationToMs(totalDuration),
			SizeBytes:      int64(len(bodyBytes)),
		},
	}, nil
}

// StreamCallback provides hooks for streaming response processing.
type StreamCallback struct {
	OnStart func(statusCode int, statusText string, headers map[string]string) error
	OnChunk func(chunk []byte) error
}

// ExecuteStream sends the request and streams the response body in chunks.
func (c *Client) ExecuteStream(ctx context.Context, req Request, cb StreamCallback) error {
	if strings.TrimSpace(req.URL) == "" {
		return errors.New("request URL cannot be empty")
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("invalid URL scheme (must be http or https): %s", req.URL)
	}

	host := parsedURL.Hostname()
	if isBlockedMetadataHost(host) {
		return fmt.Errorf("access to internal cloud metadata service is prohibited: %s", host)
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if req.Body != "" && method != http.MethodGet && method != http.MethodHead {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range req.Headers {
		if strings.EqualFold(k, "Host") {
			httpReq.Host = v
		} else {
			httpReq.Header.Set(k, v)
		}
	}

	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "APIDock/1.0 (Developer Workstation; Streaming)")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respHeaders := make(map[string]string)
	for k, values := range resp.Header {
		respHeaders[k] = strings.Join(values, ", ")
	}

	if cb.OnStart != nil {
		if err := cb.OnStart(resp.StatusCode, resp.Status, respHeaders); err != nil {
			return err
		}
	}

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 && cb.OnChunk != nil {
			if err := cb.OnChunk(buf[:n]); err != nil {
				return err
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
	}

	return nil
}

func durationToMs(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000.0
}

func isBlockedMetadataHost(host string) bool {
	blocked := []string{
		"169.254.169.254", // AWS/GCP/Azure link-local metadata
		"metadata.google.internal",
		"100.100.100.200", // Alibaba cloud
		"fd00:ec2::254",   // IPv6 AWS
	}
	for _, b := range blocked {
		if strings.EqualFold(host, b) {
			return true
		}
	}
	return false
}
