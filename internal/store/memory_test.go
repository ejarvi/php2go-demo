package store

import (
	"context"
	"errors"
	"testing"
)

type testItem struct {
	PK      string `dynamodbav:"PK"`
	SK      string `dynamodbav:"SK"`
	Name    string `dynamodbav:"name"`
	Version int    `dynamodbav:"version"`
}

func TestMemoryConditions(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()

	if err := m.Put(ctx, testItem{PK: "P", SK: "a", Name: "one", Version: 1}, IfNew); err != nil {
		t.Fatal(err)
	}
	if err := m.Put(ctx, testItem{PK: "P", SK: "a", Name: "again"}, IfNew); !errors.Is(err, ErrConflict) {
		t.Fatalf("IfNew on an existing item: got %v, want ErrConflict", err)
	}
	if err := m.Put(ctx, testItem{PK: "P", SK: "a", Name: "two", Version: 2}, IfVersion(1)); err != nil {
		t.Fatal(err)
	}
	if err := m.Put(ctx, testItem{PK: "P", SK: "a", Name: "stale", Version: 2}, IfVersion(1)); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version: got %v, want ErrConflict", err)
	}
	var got testItem
	if ok, err := m.Get(ctx, "P", "a", &got); err != nil || !ok || got.Name != "two" {
		t.Fatalf("Get: %v %v %+v", ok, err, got)
	}
}

func TestMemoryQueryAndTransact(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	for _, sk := range []string{"B#2", "A#1", "B#1"} {
		if err := m.Put(ctx, testItem{PK: "P", SK: sk}, Always); err != nil {
			t.Fatal(err)
		}
	}
	var bs []testItem
	if err := m.Query(ctx, "P", "B#", &bs); err != nil {
		t.Fatal(err)
	}
	if len(bs) != 2 || bs[0].SK != "B#1" || bs[1].SK != "B#2" {
		t.Fatalf("Query: %+v", bs)
	}

	// A failed condition changes nothing.
	err := m.Transact(ctx, DeleteOp("P", "A#1", Always), PutOp(testItem{PK: "P", SK: "B#1"}, IfNew))
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Transact: got %v, want ErrConflict", err)
	}
	if ok, _ := m.Get(ctx, "P", "A#1", &testItem{}); !ok {
		t.Fatal("a failed transaction deleted an item")
	}
}
