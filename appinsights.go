package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
)

type appInsightsHandler struct {
	client appinsights.TelemetryClient
	inner  slog.Handler
	attrs  []slog.Attr
	groups []string
}

func newAppInsightsHandler(connectionString string, inner slog.Handler) *appInsightsHandler {
	ikey, endpoint := parseConnectionString(connectionString)
	cfg := appinsights.NewTelemetryConfiguration(ikey)
	if endpoint != "" {
		cfg.EndpointUrl = endpoint + "v2/track"
	}
	client := appinsights.NewTelemetryClientFromConfig(cfg)
	return &appInsightsHandler{client: client, inner: inner}
}

func parseConnectionString(cs string) (ikey, endpoint string) {
	for _, pair := range strings.Split(cs, ";") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "InstrumentationKey":
			ikey = strings.TrimSpace(v)
		case "IngestionEndpoint":
			endpoint = strings.TrimSpace(v)
		}
	}
	return
}

func (h *appInsightsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *appInsightsHandler) Handle(ctx context.Context, r slog.Record) error {
	props := make(map[string]string)
	props["service"] = "mcp-proxy"
	for _, a := range h.attrs {
		props[a.Key] = a.Value.String()
	}

	var method, path, status, clientIP string
	var duration time.Duration
	r.Attrs(func(a slog.Attr) bool {
		key := a.Key
		if len(h.groups) > 0 {
			key = strings.Join(h.groups, ".") + "." + key
		}
		switch key {
		case "method":
			method = a.Value.String()
		case "path":
			path = a.Value.String()
		case "status":
			status = a.Value.String()
		case "duration":
			duration, _ = time.ParseDuration(a.Value.String())
		case "client_ip":
			clientIP = a.Value.String()
		}
		props[key] = a.Value.String()
		return true
	})

	if status != "" {
		req := appinsights.NewRequestTelemetry(method, path, duration, status)
		for k, v := range props {
			req.Properties[k] = v
		}
		if clientIP != "" {
			req.Tags.Location().SetIp(clientIP)
		}
		code := 0
		fmt.Sscanf(status, "%d", &code)
		req.Success = code >= 200 && code < 400
		h.client.Track(req)
	} else if r.Level >= slog.LevelError {
		exc := appinsights.NewExceptionTelemetry(r.Message)
		exc.SeverityLevel = appinsights.Error
		for k, v := range props {
			exc.Properties[k] = v
		}
		h.client.Track(exc)
	} else {
		trace := appinsights.NewTraceTelemetry(r.Message, toSeverity(r.Level))
		for k, v := range props {
			trace.Properties[k] = v
		}
		h.client.Track(trace)
	}

	return h.inner.Handle(ctx, r)
}

func (h *appInsightsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &appInsightsHandler{
		client: h.client,
		inner:  h.inner.WithAttrs(attrs),
		attrs:  append(append([]slog.Attr{}, h.attrs...), attrs...),
		groups: h.groups,
	}
}

func (h *appInsightsHandler) WithGroup(name string) slog.Handler {
	return &appInsightsHandler{
		client: h.client,
		inner:  h.inner.WithGroup(name),
		attrs:  h.attrs,
		groups: append(append([]string{}, h.groups...), name),
	}
}

func (h *appInsightsHandler) Close() {
	select {
	case <-h.client.Channel().Close(5 * time.Second):
	case <-time.After(5 * time.Second):
	}
}

func toSeverity(level slog.Level) contracts.SeverityLevel {
	switch {
	case level >= slog.LevelError:
		return appinsights.Error
	case level >= slog.LevelWarn:
		return appinsights.Warning
	case level >= slog.LevelInfo:
		return appinsights.Information
	default:
		return appinsights.Verbose
	}
}
