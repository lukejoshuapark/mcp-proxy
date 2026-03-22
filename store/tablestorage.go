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
	client *aztables.Client
}

func NewTableStorageStore[T any](serviceClient *aztables.ServiceClient, tableName string) (*TableStorageStore[T], error) {
	client := serviceClient.NewClient(tableName)

	// Ensure the table exists.
	_, err := client.CreateTable(context.Background(), nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == "TableAlreadyExists" {
			// OK — table already exists.
		} else {
			return nil, fmt.Errorf("creating table %s: %w", tableName, err)
		}
	}

	return &TableStorageStore[T]{client: client}, nil
}

func (s *TableStorageStore[T]) Get(partitionKey, sortKey string) (T, bool) {
	var zero T
	resp, err := s.client.GetEntity(context.Background(), partitionKey, sortKey, nil)
	if err != nil {
		return zero, false
	}

	var entity tableEntity
	if err := json.Unmarshal(resp.Value, &entity); err != nil {
		return zero, false
	}

	var v T
	if err := json.Unmarshal([]byte(entity.Value), &v); err != nil {
		return zero, false
	}
	return v, true
}

func (s *TableStorageStore[T]) Set(partitionKey, sortKey string, value T) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	entity := tableEntity{
		Entity: aztables.Entity{
			PartitionKey: partitionKey,
			RowKey:       sortKey,
		},
		Value: string(data),
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
	v, ok := s.Get(partitionKey, sortKey)
	if ok {
		s.Delete(partitionKey, sortKey)
	}
	return v, ok
}
