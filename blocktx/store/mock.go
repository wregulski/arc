// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package store

import (
	"context"
	"github.com/bitcoin-sv/arc/blocktx/blocktx_api"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"sync"
)

// Ensure, that InterfaceMock does implement Interface.
// If this is not the case, regenerate this file with moq.
var _ Interface = &InterfaceMock{}

// InterfaceMock is a mock implementation of Interface.
//
//	func TestSomethingThatUsesInterface(t *testing.T) {
//
//		// make and configure a mocked Interface
//		mockedInterface := &InterfaceMock{
//			CloseFunc: func() error {
//				panic("mock out the Close method")
//			},
//			GetBlockFunc: func(ctx context.Context, hash *chainhash.Hash) (*blocktx_api.Block, error) {
//				panic("mock out the GetBlock method")
//			},
//			GetBlockGapsFunc: func(ctx context.Context, heightRange int) ([]*BlockGap, error) {
//				panic("mock out the GetBlockGaps method")
//			},
//			GetPrimaryFunc: func(ctx context.Context) (string, error) {
//				panic("mock out the GetPrimary method")
//			},
//			GetTransactionBlocksFunc: func(ctx context.Context, transactions *blocktx_api.Transactions) (*blocktx_api.TransactionBlocks, error) {
//				panic("mock out the GetTransactionBlocks method")
//			},
//			GetTransactionMerklePathFunc: func(ctx context.Context, hash *chainhash.Hash) (string, error) {
//				panic("mock out the GetTransactionMerklePath method")
//			},
//			InsertBlockFunc: func(ctx context.Context, block *blocktx_api.Block) (uint64, error) {
//				panic("mock out the InsertBlock method")
//			},
//			MarkBlockAsDoneFunc: func(ctx context.Context, hash *chainhash.Hash, size uint64, txCount uint64) error {
//				panic("mock out the MarkBlockAsDone method")
//			},
//			RegisterTransactionFunc: func(ctx context.Context, transaction *blocktx_api.TransactionAndSource) error {
//				panic("mock out the RegisterTransaction method")
//			},
//			TryToBecomePrimaryFunc: func(ctx context.Context, myHostName string) error {
//				panic("mock out the TryToBecomePrimary method")
//			},
//			UpdateBlockTransactionsFunc: func(ctx context.Context, blockId uint64, transactions []*blocktx_api.TransactionAndSource, merklePaths []string) error {
//				panic("mock out the UpdateBlockTransactions method")
//			},
//		}
//
//		// use mockedInterface in code that requires Interface
//		// and then make assertions.
//
//	}
type InterfaceMock struct {
	// CloseFunc mocks the Close method.
	CloseFunc func() error

	// GetBlockFunc mocks the GetBlock method.
	GetBlockFunc func(ctx context.Context, hash *chainhash.Hash) (*blocktx_api.Block, error)

	// GetBlockGapsFunc mocks the GetBlockGaps method.
	GetBlockGapsFunc func(ctx context.Context, heightRange int) ([]*BlockGap, error)

	// GetPrimaryFunc mocks the GetPrimary method.
	GetPrimaryFunc func(ctx context.Context) (string, error)

	// GetTransactionBlocksFunc mocks the GetTransactionBlocks method.
	GetTransactionBlocksFunc func(ctx context.Context, transactions *blocktx_api.Transactions) (*blocktx_api.TransactionBlocks, error)

	// GetTransactionMerklePathFunc mocks the GetTransactionMerklePath method.
	GetTransactionMerklePathFunc func(ctx context.Context, hash *chainhash.Hash) (string, error)

	// InsertBlockFunc mocks the InsertBlock method.
	InsertBlockFunc func(ctx context.Context, block *blocktx_api.Block) (uint64, error)

	// MarkBlockAsDoneFunc mocks the MarkBlockAsDone method.
	MarkBlockAsDoneFunc func(ctx context.Context, hash *chainhash.Hash, size uint64, txCount uint64) error

	// RegisterTransactionFunc mocks the RegisterTransaction method.
	RegisterTransactionFunc func(ctx context.Context, transaction *blocktx_api.TransactionAndSource) error

	// TryToBecomePrimaryFunc mocks the TryToBecomePrimary method.
	TryToBecomePrimaryFunc func(ctx context.Context, myHostName string) error

	// UpdateBlockTransactionsFunc mocks the UpdateBlockTransactions method.
	UpdateBlockTransactionsFunc func(ctx context.Context, blockId uint64, transactions []*blocktx_api.TransactionAndSource, merklePaths []string) error

	// calls tracks calls to the methods.
	calls struct {
		// Close holds details about calls to the Close method.
		Close []struct {
		}
		// GetBlock holds details about calls to the GetBlock method.
		GetBlock []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Hash is the hash argument value.
			Hash *chainhash.Hash
		}
		// GetBlockGaps holds details about calls to the GetBlockGaps method.
		GetBlockGaps []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// HeightRange is the heightRange argument value.
			HeightRange int
		}
		// GetPrimary holds details about calls to the GetPrimary method.
		GetPrimary []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// GetTransactionBlocks holds details about calls to the GetTransactionBlocks method.
		GetTransactionBlocks []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Transactions is the transactions argument value.
			Transactions *blocktx_api.Transactions
		}
		// GetTransactionMerklePath holds details about calls to the GetTransactionMerklePath method.
		GetTransactionMerklePath []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Hash is the hash argument value.
			Hash *chainhash.Hash
		}
		// InsertBlock holds details about calls to the InsertBlock method.
		InsertBlock []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Block is the block argument value.
			Block *blocktx_api.Block
		}
		// MarkBlockAsDone holds details about calls to the MarkBlockAsDone method.
		MarkBlockAsDone []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Hash is the hash argument value.
			Hash *chainhash.Hash
			// Size is the size argument value.
			Size uint64
			// TxCount is the txCount argument value.
			TxCount uint64
		}
		// RegisterTransaction holds details about calls to the RegisterTransaction method.
		RegisterTransaction []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Transaction is the transaction argument value.
			Transaction *blocktx_api.TransactionAndSource
		}
		// TryToBecomePrimary holds details about calls to the TryToBecomePrimary method.
		TryToBecomePrimary []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// MyHostName is the myHostName argument value.
			MyHostName string
		}
		// UpdateBlockTransactions holds details about calls to the UpdateBlockTransactions method.
		UpdateBlockTransactions []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// BlockId is the blockId argument value.
			BlockId uint64
			// Transactions is the transactions argument value.
			Transactions []*blocktx_api.TransactionAndSource
			// MerklePaths is the merklePaths argument value.
			MerklePaths []string
		}
	}
	lockClose                    sync.RWMutex
	lockGetBlock                 sync.RWMutex
	lockGetBlockGaps             sync.RWMutex
	lockGetPrimary               sync.RWMutex
	lockGetTransactionBlocks     sync.RWMutex
	lockGetTransactionMerklePath sync.RWMutex
	lockInsertBlock              sync.RWMutex
	lockMarkBlockAsDone          sync.RWMutex
	lockRegisterTransaction      sync.RWMutex
	lockTryToBecomePrimary       sync.RWMutex
	lockUpdateBlockTransactions  sync.RWMutex
}

// Close calls CloseFunc.
func (mock *InterfaceMock) Close() error {
	if mock.CloseFunc == nil {
		panic("InterfaceMock.CloseFunc: method is nil but Interface.Close was just called")
	}
	callInfo := struct {
	}{}
	mock.lockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	mock.lockClose.Unlock()
	return mock.CloseFunc()
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//
//	len(mockedInterface.CloseCalls())
func (mock *InterfaceMock) CloseCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockClose.RLock()
	calls = mock.calls.Close
	mock.lockClose.RUnlock()
	return calls
}

// GetBlock calls GetBlockFunc.
func (mock *InterfaceMock) GetBlock(ctx context.Context, hash *chainhash.Hash) (*blocktx_api.Block, error) {
	if mock.GetBlockFunc == nil {
		panic("InterfaceMock.GetBlockFunc: method is nil but Interface.GetBlock was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Hash *chainhash.Hash
	}{
		Ctx:  ctx,
		Hash: hash,
	}
	mock.lockGetBlock.Lock()
	mock.calls.GetBlock = append(mock.calls.GetBlock, callInfo)
	mock.lockGetBlock.Unlock()
	return mock.GetBlockFunc(ctx, hash)
}

// GetBlockCalls gets all the calls that were made to GetBlock.
// Check the length with:
//
//	len(mockedInterface.GetBlockCalls())
func (mock *InterfaceMock) GetBlockCalls() []struct {
	Ctx  context.Context
	Hash *chainhash.Hash
} {
	var calls []struct {
		Ctx  context.Context
		Hash *chainhash.Hash
	}
	mock.lockGetBlock.RLock()
	calls = mock.calls.GetBlock
	mock.lockGetBlock.RUnlock()
	return calls
}

// GetBlockGaps calls GetBlockGapsFunc.
func (mock *InterfaceMock) GetBlockGaps(ctx context.Context, heightRange int) ([]*BlockGap, error) {
	if mock.GetBlockGapsFunc == nil {
		panic("InterfaceMock.GetBlockGapsFunc: method is nil but Interface.GetBlockGaps was just called")
	}
	callInfo := struct {
		Ctx         context.Context
		HeightRange int
	}{
		Ctx:         ctx,
		HeightRange: heightRange,
	}
	mock.lockGetBlockGaps.Lock()
	mock.calls.GetBlockGaps = append(mock.calls.GetBlockGaps, callInfo)
	mock.lockGetBlockGaps.Unlock()
	return mock.GetBlockGapsFunc(ctx, heightRange)
}

// GetBlockGapsCalls gets all the calls that were made to GetBlockGaps.
// Check the length with:
//
//	len(mockedInterface.GetBlockGapsCalls())
func (mock *InterfaceMock) GetBlockGapsCalls() []struct {
	Ctx         context.Context
	HeightRange int
} {
	var calls []struct {
		Ctx         context.Context
		HeightRange int
	}
	mock.lockGetBlockGaps.RLock()
	calls = mock.calls.GetBlockGaps
	mock.lockGetBlockGaps.RUnlock()
	return calls
}

// GetPrimary calls GetPrimaryFunc.
func (mock *InterfaceMock) GetPrimary(ctx context.Context) (string, error) {
	if mock.GetPrimaryFunc == nil {
		panic("InterfaceMock.GetPrimaryFunc: method is nil but Interface.GetPrimary was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockGetPrimary.Lock()
	mock.calls.GetPrimary = append(mock.calls.GetPrimary, callInfo)
	mock.lockGetPrimary.Unlock()
	return mock.GetPrimaryFunc(ctx)
}

// GetPrimaryCalls gets all the calls that were made to GetPrimary.
// Check the length with:
//
//	len(mockedInterface.GetPrimaryCalls())
func (mock *InterfaceMock) GetPrimaryCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockGetPrimary.RLock()
	calls = mock.calls.GetPrimary
	mock.lockGetPrimary.RUnlock()
	return calls
}

// GetTransactionBlocks calls GetTransactionBlocksFunc.
func (mock *InterfaceMock) GetTransactionBlocks(ctx context.Context, transactions *blocktx_api.Transactions) (*blocktx_api.TransactionBlocks, error) {
	if mock.GetTransactionBlocksFunc == nil {
		panic("InterfaceMock.GetTransactionBlocksFunc: method is nil but Interface.GetTransactionBlocks was just called")
	}
	callInfo := struct {
		Ctx          context.Context
		Transactions *blocktx_api.Transactions
	}{
		Ctx:          ctx,
		Transactions: transactions,
	}
	mock.lockGetTransactionBlocks.Lock()
	mock.calls.GetTransactionBlocks = append(mock.calls.GetTransactionBlocks, callInfo)
	mock.lockGetTransactionBlocks.Unlock()
	return mock.GetTransactionBlocksFunc(ctx, transactions)
}

// GetTransactionBlocksCalls gets all the calls that were made to GetTransactionBlocks.
// Check the length with:
//
//	len(mockedInterface.GetTransactionBlocksCalls())
func (mock *InterfaceMock) GetTransactionBlocksCalls() []struct {
	Ctx          context.Context
	Transactions *blocktx_api.Transactions
} {
	var calls []struct {
		Ctx          context.Context
		Transactions *blocktx_api.Transactions
	}
	mock.lockGetTransactionBlocks.RLock()
	calls = mock.calls.GetTransactionBlocks
	mock.lockGetTransactionBlocks.RUnlock()
	return calls
}

// GetTransactionMerklePath calls GetTransactionMerklePathFunc.
func (mock *InterfaceMock) GetTransactionMerklePath(ctx context.Context, hash *chainhash.Hash) (string, error) {
	if mock.GetTransactionMerklePathFunc == nil {
		panic("InterfaceMock.GetTransactionMerklePathFunc: method is nil but Interface.GetTransactionMerklePath was just called")
	}
	callInfo := struct {
		Ctx  context.Context
		Hash *chainhash.Hash
	}{
		Ctx:  ctx,
		Hash: hash,
	}
	mock.lockGetTransactionMerklePath.Lock()
	mock.calls.GetTransactionMerklePath = append(mock.calls.GetTransactionMerklePath, callInfo)
	mock.lockGetTransactionMerklePath.Unlock()
	return mock.GetTransactionMerklePathFunc(ctx, hash)
}

// GetTransactionMerklePathCalls gets all the calls that were made to GetTransactionMerklePath.
// Check the length with:
//
//	len(mockedInterface.GetTransactionMerklePathCalls())
func (mock *InterfaceMock) GetTransactionMerklePathCalls() []struct {
	Ctx  context.Context
	Hash *chainhash.Hash
} {
	var calls []struct {
		Ctx  context.Context
		Hash *chainhash.Hash
	}
	mock.lockGetTransactionMerklePath.RLock()
	calls = mock.calls.GetTransactionMerklePath
	mock.lockGetTransactionMerklePath.RUnlock()
	return calls
}

// InsertBlock calls InsertBlockFunc.
func (mock *InterfaceMock) InsertBlock(ctx context.Context, block *blocktx_api.Block) (uint64, error) {
	if mock.InsertBlockFunc == nil {
		panic("InterfaceMock.InsertBlockFunc: method is nil but Interface.InsertBlock was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Block *blocktx_api.Block
	}{
		Ctx:   ctx,
		Block: block,
	}
	mock.lockInsertBlock.Lock()
	mock.calls.InsertBlock = append(mock.calls.InsertBlock, callInfo)
	mock.lockInsertBlock.Unlock()
	return mock.InsertBlockFunc(ctx, block)
}

// InsertBlockCalls gets all the calls that were made to InsertBlock.
// Check the length with:
//
//	len(mockedInterface.InsertBlockCalls())
func (mock *InterfaceMock) InsertBlockCalls() []struct {
	Ctx   context.Context
	Block *blocktx_api.Block
} {
	var calls []struct {
		Ctx   context.Context
		Block *blocktx_api.Block
	}
	mock.lockInsertBlock.RLock()
	calls = mock.calls.InsertBlock
	mock.lockInsertBlock.RUnlock()
	return calls
}

// MarkBlockAsDone calls MarkBlockAsDoneFunc.
func (mock *InterfaceMock) MarkBlockAsDone(ctx context.Context, hash *chainhash.Hash, size uint64, txCount uint64) error {
	if mock.MarkBlockAsDoneFunc == nil {
		panic("InterfaceMock.MarkBlockAsDoneFunc: method is nil but Interface.MarkBlockAsDone was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		Hash    *chainhash.Hash
		Size    uint64
		TxCount uint64
	}{
		Ctx:     ctx,
		Hash:    hash,
		Size:    size,
		TxCount: txCount,
	}
	mock.lockMarkBlockAsDone.Lock()
	mock.calls.MarkBlockAsDone = append(mock.calls.MarkBlockAsDone, callInfo)
	mock.lockMarkBlockAsDone.Unlock()
	return mock.MarkBlockAsDoneFunc(ctx, hash, size, txCount)
}

// MarkBlockAsDoneCalls gets all the calls that were made to MarkBlockAsDone.
// Check the length with:
//
//	len(mockedInterface.MarkBlockAsDoneCalls())
func (mock *InterfaceMock) MarkBlockAsDoneCalls() []struct {
	Ctx     context.Context
	Hash    *chainhash.Hash
	Size    uint64
	TxCount uint64
} {
	var calls []struct {
		Ctx     context.Context
		Hash    *chainhash.Hash
		Size    uint64
		TxCount uint64
	}
	mock.lockMarkBlockAsDone.RLock()
	calls = mock.calls.MarkBlockAsDone
	mock.lockMarkBlockAsDone.RUnlock()
	return calls
}

// RegisterTransaction calls RegisterTransactionFunc.
func (mock *InterfaceMock) RegisterTransaction(ctx context.Context, transaction *blocktx_api.TransactionAndSource) error {
	if mock.RegisterTransactionFunc == nil {
		panic("InterfaceMock.RegisterTransactionFunc: method is nil but Interface.RegisterTransaction was just called")
	}
	callInfo := struct {
		Ctx         context.Context
		Transaction *blocktx_api.TransactionAndSource
	}{
		Ctx:         ctx,
		Transaction: transaction,
	}
	mock.lockRegisterTransaction.Lock()
	mock.calls.RegisterTransaction = append(mock.calls.RegisterTransaction, callInfo)
	mock.lockRegisterTransaction.Unlock()
	return mock.RegisterTransactionFunc(ctx, transaction)
}

// RegisterTransactionCalls gets all the calls that were made to RegisterTransaction.
// Check the length with:
//
//	len(mockedInterface.RegisterTransactionCalls())
func (mock *InterfaceMock) RegisterTransactionCalls() []struct {
	Ctx         context.Context
	Transaction *blocktx_api.TransactionAndSource
} {
	var calls []struct {
		Ctx         context.Context
		Transaction *blocktx_api.TransactionAndSource
	}
	mock.lockRegisterTransaction.RLock()
	calls = mock.calls.RegisterTransaction
	mock.lockRegisterTransaction.RUnlock()
	return calls
}

// TryToBecomePrimary calls TryToBecomePrimaryFunc.
func (mock *InterfaceMock) TryToBecomePrimary(ctx context.Context, myHostName string) error {
	if mock.TryToBecomePrimaryFunc == nil {
		panic("InterfaceMock.TryToBecomePrimaryFunc: method is nil but Interface.TryToBecomePrimary was just called")
	}
	callInfo := struct {
		Ctx        context.Context
		MyHostName string
	}{
		Ctx:        ctx,
		MyHostName: myHostName,
	}
	mock.lockTryToBecomePrimary.Lock()
	mock.calls.TryToBecomePrimary = append(mock.calls.TryToBecomePrimary, callInfo)
	mock.lockTryToBecomePrimary.Unlock()
	return mock.TryToBecomePrimaryFunc(ctx, myHostName)
}

// TryToBecomePrimaryCalls gets all the calls that were made to TryToBecomePrimary.
// Check the length with:
//
//	len(mockedInterface.TryToBecomePrimaryCalls())
func (mock *InterfaceMock) TryToBecomePrimaryCalls() []struct {
	Ctx        context.Context
	MyHostName string
} {
	var calls []struct {
		Ctx        context.Context
		MyHostName string
	}
	mock.lockTryToBecomePrimary.RLock()
	calls = mock.calls.TryToBecomePrimary
	mock.lockTryToBecomePrimary.RUnlock()
	return calls
}

// UpdateBlockTransactions calls UpdateBlockTransactionsFunc.
func (mock *InterfaceMock) UpdateBlockTransactions(ctx context.Context, blockId uint64, transactions []*blocktx_api.TransactionAndSource, merklePaths []string) error {
	if mock.UpdateBlockTransactionsFunc == nil {
		panic("InterfaceMock.UpdateBlockTransactionsFunc: method is nil but Interface.UpdateBlockTransactions was just called")
	}
	callInfo := struct {
		Ctx          context.Context
		BlockId      uint64
		Transactions []*blocktx_api.TransactionAndSource
		MerklePaths  []string
	}{
		Ctx:          ctx,
		BlockId:      blockId,
		Transactions: transactions,
		MerklePaths:  merklePaths,
	}
	mock.lockUpdateBlockTransactions.Lock()
	mock.calls.UpdateBlockTransactions = append(mock.calls.UpdateBlockTransactions, callInfo)
	mock.lockUpdateBlockTransactions.Unlock()
	return mock.UpdateBlockTransactionsFunc(ctx, blockId, transactions, merklePaths)
}

// UpdateBlockTransactionsCalls gets all the calls that were made to UpdateBlockTransactions.
// Check the length with:
//
//	len(mockedInterface.UpdateBlockTransactionsCalls())
func (mock *InterfaceMock) UpdateBlockTransactionsCalls() []struct {
	Ctx          context.Context
	BlockId      uint64
	Transactions []*blocktx_api.TransactionAndSource
	MerklePaths  []string
} {
	var calls []struct {
		Ctx          context.Context
		BlockId      uint64
		Transactions []*blocktx_api.TransactionAndSource
		MerklePaths  []string
	}
	mock.lockUpdateBlockTransactions.RLock()
	calls = mock.calls.UpdateBlockTransactions
	mock.lockUpdateBlockTransactions.RUnlock()
	return calls
}
