package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/lukejoshuapark/environment"
)

type Config struct {
	ListenAddr         string `environment:"MCP_PROXY_LISTEN_ADDR,:8080"`
	PublicURL          string `environment:"MCP_PROXY_PUBLIC_URL"`
	RemoteAuthURL      string `environment:"MCP_PROXY_REMOTE_AUTH_URL"`
	RemoteTokenURL     string `environment:"MCP_PROXY_REMOTE_TOKEN_URL"`
	RemoteClientID     string `environment:"MCP_PROXY_REMOTE_CLIENT_ID"`
	RemoteClientSecret string `environment:"MCP_PROXY_REMOTE_CLIENT_SECRET"`
	UpstreamMCPURL     string `environment:"MCP_PROXY_UPSTREAM_MCP_URL"`

	AzureStorageAccount string `environment:"MCP_PROXY_AZURE_STORAGE_ACCOUNT,"`
	AzureStorageKey     string `environment:"MCP_PROXY_AZURE_STORAGE_KEY,"`

	EncryptionKey string `environment:"MCP_PROXY_ENCRYPTION_KEY,"`
	Scopes        string `environment:"MCP_PROXY_SCOPES,"`
	Audience      string `environment:"MCP_PROXY_REMOTE_AUDIENCE,"`

	AppInsightsConnectionString string `environment:"MCP_PROXY_APPINSIGHTS_CONNECTION_STRING,"`
}

func (c Config) UseTableStorage() bool {
	return c.AzureStorageAccount != ""
}

func Load() (Config, error) {
	var cfg Config
	if err := environment.Populate(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if !strings.HasPrefix(c.PublicURL, "https://") {
		return fmt.Errorf("MCP_PROXY_PUBLIC_URL must use https scheme, got %q", c.PublicURL)
	}

	if c.EncryptionKey != "" {
		key, err := base64.RawURLEncoding.DecodeString(c.EncryptionKey)
		if err != nil || len(key) != 32 {
			return fmt.Errorf("MCP_PROXY_ENCRYPTION_KEY must be a raw-URL-base64 encoded 32-byte key for AES-256")
		}
	}

	return nil
}
