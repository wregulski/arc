package transaction_handler

import (
	"context"

	arc "github.com/bitcoin-sv/arc/api"
	"github.com/pkg/errors"
)

var (
	// ErrTransactionNotFound is returned when a transaction is not found.
	ErrTransactionNotFound = errors.New("transaction not found")

	// ErrParentTransactionNotFound is returned when a parent transaction is not found.
	ErrParentTransactionNotFound = errors.New("parent transaction not found")
)

type TransactionHandler interface {
	GetTransaction(ctx context.Context, txID string) ([]byte, error)
	GetTransactionStatus(ctx context.Context, txID string) (*TransactionStatus, error)
	SubmitTransaction(ctx context.Context, tx []byte, options *arc.TransactionOptions) (*TransactionStatus, error)
	SubmitTransactions(ctx context.Context, tx [][]byte, options *arc.TransactionOptions) ([]*TransactionStatus, error)
}

// TransactionStatus defines model for TransactionStatus.
type TransactionStatus struct {
	TxID        string
	MerklePath  string
	BlockHash   string
	BlockHeight uint64
	Status      string
	ExtraInfo   string
	Timestamp   int64
}
