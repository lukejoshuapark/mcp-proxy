package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/lukejoshuapark/mcp-proxy/config"
	"github.com/lukejoshuapark/mcp-proxy/handler"
	"github.com/lukejoshuapark/mcp-proxy/store"
)

func initializeStores(cfg config.Config) (store.Store[handler.AuthSession], store.Store[handler.StoredCode], error) {
	var encryptionKey []byte
	if cfg.EncryptionKey != "" {
		key, err := base64.RawURLEncoding.DecodeString(cfg.EncryptionKey)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing encryption key: %w", err)
		}
		encryptionKey = key
	}

	if cfg.UseTableStorage() {
		slog.Info("using azure table storage")
		cred, err := aztables.NewSharedKeyCredential(cfg.AzureStorageAccount, cfg.AzureStorageKey)
		if err != nil {
			return nil, nil, fmt.Errorf("creating azure credential: %w", err)
		}

		serviceURL := fmt.Sprintf("https://%s.table.core.windows.net", cfg.AzureStorageAccount)
		serviceClient, err := aztables.NewServiceClientWithSharedKey(serviceURL, cred, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("creating table service client: %w", err)
		}

		sessions, err := store.NewTableStorageStore[handler.AuthSession](serviceClient, "sessions", nil)
		if err != nil {
			return nil, nil, fmt.Errorf("creating sessions store: %w", err)
		}

		codes, err := store.NewTableStorageStore[handler.StoredCode](serviceClient, "codes", encryptionKey)
		if err != nil {
			return nil, nil, fmt.Errorf("creating codes store: %w", err)
		}

		return sessions, codes, nil
	}

	slog.Info("using in-memory storage")
	return store.NewInMemoryStore[handler.AuthSession](),
		store.NewInMemoryStore[handler.StoredCode](),
		nil
}
