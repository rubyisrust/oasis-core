// Package api implements the storage backend API.
package api

import (
	"context"
	"errors"

	"github.com/oasislabs/ekiden/go/common/cbor"
	"github.com/oasislabs/ekiden/go/common/crypto/hash"
	"github.com/oasislabs/ekiden/go/common/crypto/signature"
	"github.com/oasislabs/ekiden/go/common/node"
	nodedb "github.com/oasislabs/ekiden/go/storage/mkvs/urkel/db/api"
	urkelNode "github.com/oasislabs/ekiden/go/storage/mkvs/urkel/node"
	"github.com/oasislabs/ekiden/go/storage/mkvs/urkel/syncer"
	"github.com/oasislabs/ekiden/go/storage/mkvs/urkel/writelog"
)

var (
	// ErrCantProve is the error returned when the backend is incapable
	// of generating proofs (unsupported, no key, etc).
	ErrCantProve = errors.New("storage: unable to provide proofs")

	// ReceiptSignatureContext is the signature context used for verifying MKVS receipts.
	ReceiptSignatureContext = []byte("EkStrRct")

	_ cbor.Marshaler   = (*ReceiptBody)(nil)
	_ cbor.Unmarshaler = (*ReceiptBody)(nil)
	_ cbor.Marshaler   = (*Receipt)(nil)
	_ cbor.Unmarshaler = (*Receipt)(nil)
)

// WriteLog is a write log.
//
// The keys in the write log must be unique.
type WriteLog = writelog.WriteLog

// LogEntry is a write log entry.
type LogEntry = writelog.LogEntry

// WriteLogIterator iterates over write log entries.
type WriteLogIterator = nodedb.WriteLogIterator

// ReceiptBody is the body of a receipt.
type ReceiptBody struct {
	// Version is the storage data structure version.
	Version uint16
	// Roots are the merkle roots of the merklized data structure that the
	// storage node is certifying to store.
	Roots []hash.Hash
}

// Receipt is a signed ReceiptBody.
type Receipt struct {
	signature.Signed
}

// MarshalCBOR serializes the type into a CBOR byte vector.
func (rb *ReceiptBody) MarshalCBOR() []byte {
	return cbor.Marshal(rb)
}

// UnmarshalCBOR deserializes a CBOR byte vector into the given type.
func (rb *ReceiptBody) UnmarshalCBOR(data []byte) error {
	return cbor.Unmarshal(data, rb)
}

// Open first verifies the blob signature then unmarshals the blob.
func (s *Receipt) Open(receipt *ReceiptBody) error {
	return s.Signed.Open(ReceiptSignatureContext, receipt)
}

// MarshalCBOR serializes the type into a CBOR byte vector.
func (s *Receipt) MarshalCBOR() []byte {
	return s.Signed.MarshalCBOR()
}

// UnmarshalCBOR deserializes a CBOR byte vector into the given type.
func (s *Receipt) UnmarshalCBOR(data []byte) error {
	return s.Signed.UnmarshalCBOR(data)
}

// NodeID is a root-relative node identifier which uniquely identifies
// a node under a given root.
type NodeID = urkelNode.ID

// Node is either an InternalNode or a LeafNode.
type Node = urkelNode.Node

// Pointer is a pointer to another node.
type Pointer = urkelNode.Pointer

// InternalNode is an internal node with two children.
type InternalNode = urkelNode.InternalNode

// LeafNode is a leaf node containing a key/value pair.
type LeafNode = urkelNode.LeafNode

// SubtreeIndex is a subtree index.
type SubtreeIndex = syncer.SubtreeIndex

// InvalidSubtreeIndex is an invalid subtree index.
const InvalidSubtreeIndex = syncer.InvalidSubtreeIndex

// SubtreePointer is a pointer into the compressed representation of a
// subtree.
type SubtreePointer = syncer.SubtreePointer

// InternalNodeSummary is a compressed (index-only) representation of an
// internal node.
type InternalNodeSummary = syncer.InternalNodeSummary

// Subtree is a compressed representation of a subtree.
type Subtree = syncer.Subtree

// ApplyOp is an apply operation within a batch of apply operations.
type ApplyOp struct {
	// Root is the merkle root to apply the operations against. It may
	// refer to a nil node (empty hash) in which case a new root will be
	// created.
	Root hash.Hash
	// ExpectedNewRoot is the expected merkle root after applying the
	// write log.
	ExpectedNewRoot hash.Hash
	// WriteLog is a write log of operations to apply.
	WriteLog WriteLog
}

// Backend is a storage backend implementation.
type Backend interface {
	syncer.ReadSyncer

	// Apply applies a set of operations against the MKVS.  The root may refer
	// to a nil node, in which case a new root will be created.
	// The expected new root is used to check if the new root after all the
	// operations are applied already exists in the local DB.  If it does, the
	// Apply is ignored.
	Apply(context.Context, hash.Hash, hash.Hash, WriteLog) ([]*Receipt, error)

	// ApplyBatch applies multiple sets of operations against the MKVS and
	// returns a single receipt covering all applied roots.
	//
	// See Apply for more details.
	ApplyBatch(context.Context, []ApplyOp) ([]*Receipt, error)

	// GetDiff returns an iterator of write log entries that must be applied
	// to get from the first given hash to the second one.
	GetDiff(context.Context, hash.Hash, hash.Hash) (WriteLogIterator, error)

	// Cleanup closes/cleans up the storage backend.
	Cleanup()

	// Initialized returns a channel that will be closed when the
	// backend is initialized and ready to service requests.
	Initialized() <-chan struct{}
}

// ClientBackend is a storage client backend implementation.
type ClientBackend interface {
	Backend

	// GetConnectedNodes returns currently connected storage nodes.
	GetConnectedNodes() []*node.Node
}

// Genesis is the storage genesis state.
type Genesis struct {
	// State is the genesis state for the merklized key-value store.
	State WriteLog `codec:"state"`
}
