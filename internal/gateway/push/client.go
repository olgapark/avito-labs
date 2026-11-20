package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	endpoint  string
	client    *http.Client
	transport *http.Transport
}

type StatusError struct{ Code int }

func (e *StatusError) Error() string {
	return fmt.Sprintf("push service returned HTTP %d", e.Code)
}

func New(baseURL *url.URL, timeout time.Duration, traces trace.TracerProvider) *Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, DialContext: dialer.DialContext, ForceAttemptHTTP2: true,
		TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout,
	}
	return &Client{
		endpoint:  baseURL.JoinPath("/api/v1/push").String(),
		transport: transport,
		client: &http.Client{
			Timeout: timeout,
			Transport: otelhttp.NewTransport(transport, otelhttp.WithTracerProvider(traces),
				otelhttp.WithPropagators(propagation.TraceContext{}), otelhttp.WithMeterProvider(noop.NewMeterProvider()),
				otelhttp.WithSpanNameFormatter(func(_ string, _ *http.Request) string { return "POST /api/v1/push" })),
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *Client) RequestPosition(ctx context.Context, tripID, driverID uuid.UUID) error {
	payload := struct {
		RequestID   string            `json:"request_id"`
		RecipientID uuid.UUID         `json:"recipient_id"`
		Kind        string            `json:"kind"`
		Data        map[string]string `json:"data"`
	}{
		RequestID: uuid.NewString(), RecipientID: driverID, Kind: "REQUEST_POSITION",
		Data: map[string]string{"trip_id": tripID.String()},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode push request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send push request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusAccepted {
		return &StatusError{Code: resp.StatusCode}
	}
	return nil
}

func (c *Client) Close() { c.transport.CloseIdleConnections() }
