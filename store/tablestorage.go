package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
)

type tableEntity struct {
	aztables.Entity
	Value string `json:"Value"`
}

type TableStorageStore[T any] struct {
	client        *aztables.Client
	encryptionKey []byte
}

func NewTableStorageStore[T any](serviceClient *aztables.ServiceClient, tableName string, encryptionKey []byte) (*TableStorageStore[T], error) {
	client := serviceClient.NewClient(tableName)

	_, err := client.CreateTable(context.Background(), nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == "TableAlreadyExists" {
		} else {
			return nil, fmt.Errorf("creating table %s: %w", tableName, err)
		}
	}

	return &TableStorageStore[T]{client: client, encryptionKey: encryptionKey}, nil
}

func (s *TableStorageStore[T]) getEntity(partitionKey, sortKey string) (T, azcore.ETag, bool) {
	var zero T
	resp, err := s.client.GetEntity(context.Background(), partitionKey, sortKey, nil)
	if err != nil {
		return zero, "", false
	}

	var entity tableEntity
	if err := json.Unmarshal(resp.Value, &entity); err != nil {
		return zero, "", false
	}

	value := entity.Value
	if s.encryptionKey != nil {
		decrypted, err := Decrypt(value, s.encryptionKey)
		if err != nil {
			return zero, "", false
		}
		value = string(decrypted)
	}

	var v T
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		return zero, "", false
	}
	return v, resp.ETag, true
}

func (s *TableStorageStore[T]) Get(partitionKey, sortKey string) (T, bool) {
	v, _, ok := s.getEntity(partitionKey, sortKey)
	return v, ok
}

func (s *TableStorageStore[T]) Set(partitionKey, sortKey string, value T) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	valueStr := string(data)
	if s.encryptionKey != nil {
		encrypted, err := Encrypt(data, s.encryptionKey)
		if err != nil {
			return
		}
		valueStr = encrypted
	}

	entity := tableEntity{
		Entity: aztables.Entity{
			PartitionKey: partitionKey,
			RowKey:       sortKey,
		},
		Value: valueStr,
	}

	raw, err := json.Marshal(entity)
	if err != nil {
		return
	}

	s.client.UpsertEntity(context.Background(), raw, &aztables.UpsertEntityOptions{
		UpdateMode: aztables.UpdateModeReplace,
	})
}

func (s *TableStorageStore[T]) Delete(partitionKey, sortKey string) {
	s.client.DeleteEntity(context.Background(), partitionKey, sortKey, nil)
}

func (s *TableStorageStore[T]) Pop(partitionKey, sortKey string) (T, bool) {
	v, etag, ok := s.getEntity(partitionKey, sortKey)
	if !ok {
		var zero T
		return zero, false
	}

	_, err := s.client.DeleteEntity(context.Background(), partitionKey, sortKey, &aztables.DeleteEntityOptions{
		IfMatch: &etag,
	})
	if err != nil {
		var zero T
		return zero, false
	}
	return v, true
}
