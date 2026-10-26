// Package store is the only code that talks to storage. It has two implementations of
// one interface: DynamoDB on AWS (dynamo.go) and in memory for tests and local runs
// (memory.go). Items are Go structs with PK and SK fields, converted with the AWS SDK's
// attributevalue package, so `dynamodbav` struct tags work the same in both.
package store

import (
	"context"
	"errors"
)

// Store holds the site's items, all in one table keyed by PK (partition) and SK (sort).
type Store interface {
	// Get reads one item into out (a pointer to a struct), and reports false if there's none.
	Get(ctx context.Context, pk, sk string, out any) (bool, error)
	// Put writes item (a struct with PK and SK fields) if cond holds, or returns ErrConflict.
	Put(ctx context.Context, item any, cond Condition) error
	// Query reads every item in partition pk whose SK starts with prefix ("" for all),
	// in SK order, into out (a pointer to a slice of structs).
	Query(ctx context.Context, pk, prefix string, out any) error
	// Delete removes one item. Deleting an item that doesn't exist is not an error.
	Delete(ctx context.Context, pk, sk string) error
	// Transact applies all ops or none. If any op's condition fails, it returns ErrConflict
	// and changes nothing. At most MaxOps ops, and one op per key.
	Transact(ctx context.Context, ops ...Op) error
	// Ping checks that storage can be read, for the health check (D-021).
	Ping(ctx context.Context) error
}

// MaxOps is the most ops one transaction may hold (DynamoDB's limit).
const MaxOps = 100

// ErrConflict means a write's condition wasn't met: someone else changed the item first.
var ErrConflict = errors.New("store: the item changed; try again")

// Op is one change in a transaction: a Put (Item set) or a Delete (PK and SK set).
type Op struct {
	Item   any
	PK, SK string
	Cond   Condition
}

// PutOp and DeleteOp build transaction ops.
func PutOp(item any, cond Condition) Op         { return Op{Item: item, Cond: cond} }
func DeleteOp(pk, sk string, cond Condition) Op { return Op{PK: pk, SK: sk, Cond: cond} }

// Condition controls when a write may happen. Use Always, IfNew or IfVersion.
type Condition struct {
	kind    condKind
	version int
}

type condKind int

const (
	condAlways condKind = iota
	condNew
	condVersion
)

// VersionAttr is the attribute IfVersion checks. Items that use optimistic locking
// carry it, and every write of such an item increments it.
const VersionAttr = "version"

var (
	Always = Condition{kind: condAlways} // whatever is there
	IfNew  = Condition{kind: condNew}    // only if no item with this key exists
)

// IfVersion allows the write only if the stored item's version is v. With v = 0 it also
// allows the write when there's no item yet.
func IfVersion(v int) Condition { return Condition{kind: condVersion, version: v} }
