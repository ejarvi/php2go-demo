package store

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Memory is an in-memory Store for tests and local runs. Items are stored in their
// DynamoDB form, so struct tags and conditions behave as they do on AWS.
type Memory struct {
	mu    sync.Mutex
	items map[string]map[string]map[string]types.AttributeValue
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{items: map[string]map[string]map[string]types.AttributeValue{}}
}

func (m *Memory) Get(_ context.Context, pk, sk string, out any) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[pk][sk]
	if !ok {
		return false, nil
	}
	return true, attributevalue.UnmarshalMap(item, out)
}

func (m *Memory) Put(_ context.Context, item any, cond Condition) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	pk, sk, err := keyOf(av)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.holds(pk, sk, cond) {
		return ErrConflict
	}
	m.set(pk, sk, av)
	return nil
}

func (m *Memory) Query(_ context.Context, pk, prefix string, out any) error {
	m.mu.Lock()
	keys := make([]string, 0, len(m.items[pk]))
	for sk := range m.items[pk] {
		if strings.HasPrefix(sk, prefix) {
			keys = append(keys, sk)
		}
	}
	sort.Strings(keys)
	list := make([]map[string]types.AttributeValue, 0, len(keys))
	for _, sk := range keys {
		list = append(list, m.items[pk][sk])
	}
	m.mu.Unlock()
	return attributevalue.UnmarshalListOfMaps(list, out)
}

func (m *Memory) Delete(_ context.Context, pk, sk string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items[pk], sk)
	return nil
}

func (m *Memory) Transact(_ context.Context, ops ...Op) error {
	if len(ops) > MaxOps {
		return fmt.Errorf("store: %d ops in one transaction, at most %d", len(ops), MaxOps)
	}
	type put struct {
		pk, sk string
		av     map[string]types.AttributeValue
	}
	puts := make([]put, len(ops))
	for i, op := range ops {
		if op.Item == nil {
			puts[i] = put{pk: op.PK, sk: op.SK}
			continue
		}
		av, err := attributevalue.MarshalMap(op.Item)
		if err != nil {
			return err
		}
		pk, sk, err := keyOf(av)
		if err != nil {
			return err
		}
		puts[i] = put{pk, sk, av}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, op := range ops {
		if !m.holds(puts[i].pk, puts[i].sk, op.Cond) {
			return ErrConflict
		}
	}
	for _, p := range puts {
		if p.av == nil {
			delete(m.items[p.pk], p.sk)
		} else {
			m.set(p.pk, p.sk, p.av)
		}
	}
	return nil
}

func (m *Memory) Ping(context.Context) error { return nil }

func (m *Memory) set(pk, sk string, av map[string]types.AttributeValue) {
	if m.items[pk] == nil {
		m.items[pk] = map[string]map[string]types.AttributeValue{}
	}
	m.items[pk][sk] = av
}

// holds reports whether cond is met for the item at pk/sk. The caller holds the lock.
func (m *Memory) holds(pk, sk string, cond Condition) bool {
	item, exists := m.items[pk][sk]
	switch cond.kind {
	case condNew:
		return !exists
	case condVersion:
		if !exists {
			return cond.version == 0
		}
		return versionOf(item) == cond.version
	}
	return true
}

func versionOf(item map[string]types.AttributeValue) int {
	n, ok := item[VersionAttr].(*types.AttributeValueMemberN)
	if !ok {
		return 0
	}
	v, _ := strconv.Atoi(n.Value)
	return v
}

// keyOf returns an item's PK and SK, which every item must have as strings.
func keyOf(av map[string]types.AttributeValue) (string, string, error) {
	pk, ok1 := av["PK"].(*types.AttributeValueMemberS)
	sk, ok2 := av["SK"].(*types.AttributeValueMemberS)
	if !ok1 || !ok2 || pk.Value == "" || sk.Value == "" {
		return "", "", fmt.Errorf("store: item needs string PK and SK")
	}
	return pk.Value, sk.Value, nil
}
