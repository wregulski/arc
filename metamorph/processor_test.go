package metamorph_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bitcoin-sv/arc/blocktx/blocktx_api"
	. "github.com/bitcoin-sv/arc/metamorph"
	"github.com/bitcoin-sv/arc/metamorph/metamorph_api"
	. "github.com/bitcoin-sv/arc/metamorph/mocks"
	"github.com/bitcoin-sv/arc/metamorph/processor_response"
	"github.com/bitcoin-sv/arc/metamorph/store"
	"github.com/bitcoin-sv/arc/metamorph/store/badger"
	"github.com/bitcoin-sv/arc/metamorph/store/sqlite"
	"github.com/bitcoin-sv/arc/testdata"
	"github.com/labstack/gommon/random"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-p2p"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate moq -pkg mocks -out ./mocks/store_mock.go ./store/ MetamorphStore
//go:generate moq -pkg mocks -out ./mocks/blocktx_mock.go ../blocktx/ ClientI

func TestNewProcessor(t *testing.T) {
	mtmStore := &MetamorphStoreMock{
		GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
			return &store.StoreData{Hash: testdata.TX2Hash}, nil
		},
		SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error { return nil },
		RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
			return nil
		},
	}

	pm := p2p.NewPeerManagerMock()

	tt := []struct {
		name  string
		store store.MetamorphStore
		pm    p2p.PeerManagerI

		expectedErrorStr        string
		expectedNonNilProcessor bool
	}{
		{
			name:                    "success",
			store:                   mtmStore,
			pm:                      pm,
			expectedNonNilProcessor: true,
		},
		{
			name:  "no store",
			store: nil,
			pm:    pm,

			expectedErrorStr: "store cannot be nil",
		},
		{
			name:  "no pm",
			store: mtmStore,
			pm:    nil,

			expectedErrorStr: "peer manager cannot be nil",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			processor, err := NewProcessor(tc.store, tc.pm, nil,
				WithCacheExpiryTime(time.Second*5),
				WithProcessCheckIfMinedInterval(time.Second*5),
				WithProcessorLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: LogLevelDefault}))),
			)
			if tc.expectedErrorStr != "" || err != nil {
				require.ErrorContains(t, err, tc.expectedErrorStr)
				return
			} else {
				require.NoError(t, err)
			}

			defer processor.Shutdown()

			if tc.expectedNonNilProcessor && processor == nil {
				t.Error("Expected a non-nil Processor")
			}
		})
	}
}

func TestLoadUnmined(t *testing.T) {
	storedAt := time.Date(2023, 10, 3, 5, 0, 0, 0, time.UTC)

	tt := []struct {
		name              string
		storedData        []*store.StoreData
		transactionBlocks *blocktx_api.TransactionBlocks
		maxMonitoredTxs   int64
		getUnminedErr     error

		expectedGetTransactionBlocksCalls int
		expectedItemTxHashesFinal         []*chainhash.Hash
	}{
		{
			name: "no unmined transactions loaded",

			expectedGetTransactionBlocksCalls: 0,
			expectedItemTxHashesFinal:         []*chainhash.Hash{testdata.TX3Hash, testdata.TX4Hash},
		},
		{
			name: "load 2 unmined transactions, none mined",
			storedData: []*store.StoreData{
				{
					StoredAt:    storedAt,
					AnnouncedAt: storedAt.Add(1 * time.Second),
					Hash:        testdata.TX1Hash,
					Status:      metamorph_api.Status_ANNOUNCED_TO_NETWORK,
				},
				{
					StoredAt:    storedAt,
					AnnouncedAt: storedAt.Add(1 * time.Second),
					Hash:        testdata.TX2Hash,
					Status:      metamorph_api.Status_STORED,
				},
			},
			transactionBlocks: &blocktx_api.TransactionBlocks{TransactionBlocks: []*blocktx_api.TransactionBlock{{
				BlockHash:       nil,
				BlockHeight:     0,
				TransactionHash: nil,
			}}},
			maxMonitoredTxs: 5,

			expectedGetTransactionBlocksCalls: 1,
			expectedItemTxHashesFinal:         []*chainhash.Hash{testdata.TX1Hash, testdata.TX2Hash, testdata.TX3Hash, testdata.TX4Hash},
		},
		{
			name:            "load 2 unmined transactions, failed to get unmined",
			storedData:      []*store.StoreData{},
			getUnminedErr:   errors.New("failed to get unmined"),
			maxMonitoredTxs: 5,

			expectedItemTxHashesFinal: []*chainhash.Hash{testdata.TX3Hash, testdata.TX4Hash},
		},
		{
			name: "load 2 unmined transactions, limit reached",
			storedData: []*store.StoreData{
				{
					StoredAt:    storedAt,
					AnnouncedAt: storedAt.Add(1 * time.Second),
					Hash:        testdata.TX1Hash,
					Status:      metamorph_api.Status_ANNOUNCED_TO_NETWORK,
				},
				{
					StoredAt:    storedAt,
					AnnouncedAt: storedAt.Add(1 * time.Second),
					Hash:        testdata.TX2Hash,
					Status:      metamorph_api.Status_STORED,
				},
			},
			maxMonitoredTxs: 1,

			expectedGetTransactionBlocksCalls: 0,
			expectedItemTxHashesFinal:         []*chainhash.Hash{testdata.TX3Hash, testdata.TX4Hash},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pm := p2p.NewPeerManagerMock()

			mtmStore := &MetamorphStoreMock{
				GetUnminedFunc: func(ctx context.Context, since time.Time, limit int64) ([]*store.StoreData, error) {
					return tc.storedData, tc.getUnminedErr
				},
				SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error {
					require.ElementsMatch(t, tc.expectedItemTxHashesFinal, hashes)
					require.Equal(t, len(tc.expectedItemTxHashesFinal), len(hashes))
					return nil
				},
				UpdateMinedFunc: func(ctx context.Context, hash *chainhash.Hash, blockHash *chainhash.Hash, blockHeight uint64) error {
					return nil
				},
			}

			btc := &ClientIMock{GetTransactionBlocksFunc: func(ctx context.Context, transaction *blocktx_api.Transactions) (*blocktx_api.TransactionBlocks, error) {
				return nil, nil
			}}

			processor, err := NewProcessor(mtmStore, pm, btc,
				WithProcessCheckIfMinedInterval(time.Hour*24),
				WithCacheExpiryTime(time.Hour*24),
				WithNow(func() time.Time {
					return storedAt.Add(1 * time.Hour)
				}),
				WithDataRetentionPeriod(time.Hour*24),
				WithMaxMonitoredTxs(tc.maxMonitoredTxs),
			)
			require.NoError(t, err)
			defer processor.Shutdown()
			require.Equal(t, 0, processor.ProcessorResponseMap.Len())

			processor.ProcessorResponseMap.Set(testdata.TX3Hash, processor_response.NewProcessorResponseWithStatus(testdata.TX3Hash, metamorph_api.Status_ANNOUNCED_TO_NETWORK))
			processor.ProcessorResponseMap.Set(testdata.TX4Hash, processor_response.NewProcessorResponseWithStatus(testdata.TX4Hash, metamorph_api.Status_ANNOUNCED_TO_NETWORK))

			processor.LoadUnmined()

			time.Sleep(time.Millisecond * 200)

			allItemHashes := make([]*chainhash.Hash, 0, len(processor.ProcessorResponseMap.Items()))

			require.Equal(t, tc.expectedGetTransactionBlocksCalls, len(btc.GetTransactionBlocksCalls()))

			for i, item := range processor.ProcessorResponseMap.Items() {
				require.Equal(t, i, *item.Hash)
				allItemHashes = append(allItemHashes, item.Hash)
			}

			require.ElementsMatch(t, tc.expectedItemTxHashesFinal, allItemHashes)
		})
	}
}

func TestProcessTransaction(t *testing.T) {
	tt := []struct {
		name            string
		storeData       *store.StoreData
		storeDataGetErr error

		expectedResponseMapItems int
		expectedResponses        []metamorph_api.Status
		expectedSetCalls         int
	}{
		{
			name:            "record not found",
			storeData:       nil,
			storeDataGetErr: store.ErrNotFound,

			expectedResponses: []metamorph_api.Status{
				metamorph_api.Status_RECEIVED,
				metamorph_api.Status_STORED,
				metamorph_api.Status_ANNOUNCED_TO_NETWORK,
			},
			expectedResponseMapItems: 1,
			expectedSetCalls:         1,
		},
		{
			name: "record found",
			storeData: &store.StoreData{
				Hash:         testdata.TX1Hash,
				Status:       metamorph_api.Status_REJECTED,
				RejectReason: "missing inputs",
			},
			storeDataGetErr: nil,

			expectedResponses: []metamorph_api.Status{
				metamorph_api.Status_REJECTED,
			},
			expectedSetCalls: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					require.Equal(t, testdata.TX1Hash[:], key)

					return tc.storeData, tc.storeDataGetErr
				},
				SetFunc: func(ctx context.Context, key []byte, value *store.StoreData) error {
					require.Equal(t, testdata.TX1Hash[:], key)

					return nil
				},
				UpdateStatusFunc: func(ctx context.Context, hash *chainhash.Hash, status metamorph_api.Status, rejectReason string) error {
					require.Equal(t, testdata.TX1Hash, hash)

					return nil
				},
			}
			pm := p2p.NewPeerManagerMock()

			btc := &ClientIMock{
				RegisterTransactionFunc: func(ctx context.Context, transaction *blocktx_api.TransactionAndSource) error {
					return nil
				},
			}

			processor, err := NewProcessor(s, pm, btc)
			require.NoError(t, err)
			require.Equal(t, 0, processor.ProcessorResponseMap.Len())

			responseChannel := make(chan processor_response.StatusAndError)

			var wg sync.WaitGroup
			wg.Add(len(tc.expectedResponses))
			go func() {
				n := 0
				for response := range responseChannel {
					status := response.Status
					require.Equal(t, testdata.TX1Hash, response.Hash)
					require.Equalf(t, tc.expectedResponses[n], status, "Iteration %d: Expected %s, got %s", n, tc.expectedResponses[n].String(), status.String())
					wg.Done()
					n++
				}
			}()

			processor.ProcessTransaction(context.TODO(), &ProcessorRequest{
				Data: &store.StoreData{
					Hash: testdata.TX1Hash,
				},
				ResponseChannel: responseChannel,
			})
			wg.Wait()

			if tc.expectedResponseMapItems > 0 {
				require.Equal(t, tc.expectedResponseMapItems, processor.ProcessorResponseMap.Len())
				items := processor.ProcessorResponseMap.Items()
				require.Equal(t, testdata.TX1Hash, items[*testdata.TX1Hash].Hash)
				require.Equal(t, metamorph_api.Status_ANNOUNCED_TO_NETWORK, items[*testdata.TX1Hash].Status)

				require.Len(t, pm.AnnouncedTransactions, 1)
				require.Equal(t, testdata.TX1Hash, pm.AnnouncedTransactions[0])
			}

			require.Equal(t, tc.expectedSetCalls, len(s.SetCalls()))
		})
	}
}

func Benchmark_ProcessTransaction(b *testing.B) {
	s, err := sqlite.New(true, "") // prevents profiling database code
	require.NoError(b, err)

	pm := p2p.NewPeerManagerMock()

	processor, err := NewProcessor(s, pm, nil)
	require.NoError(b, err)
	assert.Equal(b, 0, processor.ProcessorResponseMap.Len())

	btTx, _ := bt.NewTxFromBytes(testdata.TX1RawBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		btTx.Inputs[0].SequenceNumber = uint32(i)
		hash, _ := chainhash.NewHashFromStr(btTx.TxID())
		processor.ProcessTransaction(context.TODO(), &ProcessorRequest{
			Data: &store.StoreData{
				Hash: hash,
			},
		})
	}
}

func TestSendStatusForTransaction(t *testing.T) {
	tt := []struct {
		name                string
		updateStatus        metamorph_api.Status
		txResponseHash      *chainhash.Hash
		txResponseHashValue *processor_response.ProcessorResponse
		statusErr           error
		updateErr           error

		expectedUpdateStatusCalls int
		expectedStatusUpdated     bool
	}{
		{
			name:         "tx not in response map - no update",
			updateStatus: metamorph_api.Status_ANNOUNCED_TO_NETWORK,

			expectedUpdateStatusCalls: 0,
		},
		{
			name:                "tx in response map - current status REJECTED, new status SEEN_ON_NETWORK - no update",
			updateStatus:        metamorph_api.Status_SEEN_ON_NETWORK,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_REJECTED),

			expectedUpdateStatusCalls: 0,
		},
		{
			name:                "new status ANNOUNCED_TO_NETWORK - update",
			updateStatus:        metamorph_api.Status_ANNOUNCED_TO_NETWORK,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_STORED),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status REJECTED - update",
			updateStatus:        metamorph_api.Status_REJECTED,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_SENT_TO_NETWORK),
			statusErr:           errors.New("missing inputs"),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status SEEN_ON_NETWORK - update",
			updateStatus:        metamorph_api.Status_SEEN_ON_NETWORK,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_SENT_TO_NETWORK),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status ACCEPTED_BY_NETWORK - update",
			updateStatus:        metamorph_api.Status_ACCEPTED_BY_NETWORK,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_SENT_TO_NETWORK),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status SENT_TO_NETWORK - update",
			updateStatus:        metamorph_api.Status_SENT_TO_NETWORK,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_REQUESTED_BY_NETWORK),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status REQUESTED_BY_NETWORK - update",
			updateStatus:        metamorph_api.Status_REQUESTED_BY_NETWORK,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_STORED),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status MINED - update",
			updateStatus:        metamorph_api.Status_MINED,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_SEEN_ON_NETWORK),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
		{
			name:                "new status MINED - update error",
			updateStatus:        metamorph_api.Status_MINED,
			txResponseHash:      testdata.TX1Hash,
			txResponseHashValue: processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_SEEN_ON_NETWORK),
			updateErr:           errors.New("failed to update status"),

			expectedUpdateStatusCalls: 1,
			expectedStatusUpdated:     true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(tc.expectedUpdateStatusCalls)
			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					return &store.StoreData{Hash: testdata.TX2Hash}, nil
				},
				UpdateStatusFunc: func(ctx context.Context, hash *chainhash.Hash, status metamorph_api.Status, rejectReason string) error {
					require.Equal(t, tc.txResponseHash, hash)
					wg.Done()
					return tc.updateErr
				},
				SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error {
					return nil
				},
			}

			pm := p2p.NewPeerManagerMock()

			processor, err := NewProcessor(metamorphStore, pm, nil, WithNow(func() time.Time {
				return time.Date(2023, 10, 1, 13, 0, 0, 0, time.UTC)
			}))
			require.NoError(t, err)
			assert.Equal(t, 0, processor.ProcessorResponseMap.Len())

			if tc.txResponseHash != nil {
				processor.ProcessorResponseMap.Set(tc.txResponseHash, tc.txResponseHashValue)
			}

			statusUpdated, sendErr := processor.SendStatusForTransaction(testdata.TX1Hash, tc.updateStatus, "test", tc.statusErr)
			assert.NoError(t, sendErr)
			assert.Equal(t, tc.expectedStatusUpdated, statusUpdated)

			if waitTimeout(wg, time.Millisecond*200) {
				t.Fatal("status was not updated as expected")
			}

			assert.Equal(t, tc.expectedUpdateStatusCalls, len(metamorphStore.UpdateStatusCalls()))
			processor.Shutdown()
		})
	}
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func TestSendStatusMinedForTransaction(t *testing.T) {
	t.Run("SendStatusMinedForTransaction known tx", func(t *testing.T) {
		s, err := sqlite.New(true, "")
		require.NoError(t, err)
		setStoreTestData(t, s)

		pm := p2p.NewPeerManagerMock()

		processor, err := NewProcessor(s, pm, nil)
		require.NoError(t, err)
		processor.ProcessorResponseMap.Set(testdata.TX1Hash, processor_response.NewProcessorResponseWithStatus(
			testdata.TX1Hash,
			metamorph_api.Status_SEEN_ON_NETWORK,
		))
		assert.Equal(t, 1, processor.ProcessorResponseMap.Len())

		ok, sendErr := processor.SendStatusMinedForTransaction(testdata.TX1Hash, testdata.Block1Hash, 1233)
		time.Sleep(100 * time.Millisecond)
		assert.True(t, ok)
		assert.NoError(t, sendErr)
		assert.Equal(t, 0, processor.ProcessorResponseMap.Len())

		txStored, err := s.Get(context.Background(), testdata.TX1Hash[:])
		require.NoError(t, err)
		assert.Equal(t, metamorph_api.Status_MINED, txStored.Status)
	})

	t.Run("SendStatusForTransaction known tx - processed", func(t *testing.T) {
		s, err := sqlite.New(true, "")
		require.NoError(t, err)

		pm := p2p.NewPeerManagerMock()

		btc := &ClientIMock{
			RegisterTransactionFunc: func(ctx context.Context, transaction *blocktx_api.TransactionAndSource) error {
				return nil
			},
		}

		processor, err := NewProcessor(s, pm, btc)
		require.NoError(t, err)
		assert.Equal(t, 0, processor.ProcessorResponseMap.Len())

		responseChannel := make(chan processor_response.StatusAndError)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			for response := range responseChannel {
				status := response.Status
				fmt.Printf("response: %s\n", status)
				if status == metamorph_api.Status_ANNOUNCED_TO_NETWORK {
					close(responseChannel)
					time.Sleep(1 * time.Millisecond)
					wg.Done()
					return
				}
			}
		}()

		processor.ProcessTransaction(context.TODO(), &ProcessorRequest{
			Data: &store.StoreData{
				Hash: testdata.TX1Hash,
			},
			ResponseChannel: responseChannel,
		})
		wg.Wait()

		assert.Equal(t, 1, processor.ProcessorResponseMap.Len())

		ok, sendErr := processor.SendStatusMinedForTransaction(testdata.TX1Hash, testdata.Block1Hash, 1233)
		time.Sleep(10 * time.Millisecond)
		assert.True(t, ok)
		assert.NoError(t, sendErr)
		assert.Equal(t, 0, processor.ProcessorResponseMap.Len(), "should have been removed from the map")

		txStored, err := s.Get(context.Background(), testdata.TX1Hash[:])
		require.NoError(t, err)
		assert.Equal(t, metamorph_api.Status_MINED, txStored.Status)
	})
}

func BenchmarkProcessTransaction(b *testing.B) {
	direName := fmt.Sprintf("./test-benchmark-%s", random.String(6))
	s, err := badger.New(direName)
	require.NoError(b, err)
	defer func() {
		_ = s.Close(context.Background())
		_ = os.RemoveAll(direName)
	}()

	pm := p2p.NewPeerManagerMock()
	processor, err := NewProcessor(s, pm, nil)
	require.NoError(b, err)
	assert.Equal(b, 0, processor.ProcessorResponseMap.Len())

	txs := make(map[string]*chainhash.Hash)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txID := fmt.Sprintf("%x", i)

		txHash := chainhash.HashH([]byte(txID))

		txs[txID] = &txHash

		processor.ProcessTransaction(context.TODO(), &ProcessorRequest{
			Data: &store.StoreData{
				Hash:   &txHash,
				Status: metamorph_api.Status_UNKNOWN,
				RawTx:  testdata.TX1RawBytes,
			},
		})
	}
	b.StopTimer()

	// wait for the last ResponseItems to be written to the store
	time.Sleep(1 * time.Second)
}

func TestProcessCheckIfMined(t *testing.T) {
	txsBlocks := []*blocktx_api.TransactionBlock{
		{
			BlockHash:       testdata.Block1Hash[:],
			BlockHeight:     1234,
			TransactionHash: testdata.TX1Hash[:],
		},
		{
			BlockHash:       testdata.Block1Hash[:],
			BlockHeight:     1234,
			TransactionHash: testdata.TX2Hash[:],
		},
		{
			BlockHash:       testdata.Block1Hash[:],
			BlockHeight:     1234,
			TransactionHash: testdata.TX3Hash[:],
		},
	}

	tt := []struct {
		name                    string
		blocks                  []*blocktx_api.TransactionBlock
		getTransactionBlocksErr error
		updateMinedErr          error

		expectedNrOfUpdates int
	}{
		{
			name:   "expired txs",
			blocks: txsBlocks,

			expectedNrOfUpdates: 3,
		},
		{
			name:                    "failed to get transaction blocks",
			getTransactionBlocksErr: errors.New("failed to get transaction blocks"),

			expectedNrOfUpdates: 0,
		},
		{
			name: "failed to parse block hash",
			blocks: []*blocktx_api.TransactionBlock{{
				BlockHash: []byte("not a valid block hash"),
			}},

			expectedNrOfUpdates: 0,
		},
		{
			name:           "failed to update mined",
			blocks:         txsBlocks,
			updateMinedErr: errors.New("failed to update mined"),

			expectedNrOfUpdates: 3,
		},
		{
			name: "failed to get tx from response map",
			blocks: []*blocktx_api.TransactionBlock{
				{
					BlockHash:       testdata.Block1Hash[:],
					BlockHeight:     1234,
					TransactionHash: testdata.TX5Hash[:],
				},
			},

			expectedNrOfUpdates: 0,
		},
		{
			name: "missing block info",
			blocks: []*blocktx_api.TransactionBlock{
				{
					TransactionHash: testdata.TX1Hash[:],
				},
				{
					TransactionHash: testdata.TX2Hash[:],
				},
				{
					TransactionHash: testdata.TX3Hash[:],
				},
			},

			expectedNrOfUpdates: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					return &store.StoreData{Hash: testdata.TX2Hash}, nil
				},
				UpdateMinedFunc: func(ctx context.Context, hash *chainhash.Hash, blockHash *chainhash.Hash, blockHeight uint64) error {
					require.Condition(t, func() (success bool) {
						oneOfHash := hash.IsEqual(testdata.TX1Hash) || hash.IsEqual(testdata.TX2Hash) || hash.IsEqual(testdata.TX3Hash)
						return oneOfHash
					})

					return tc.updateMinedErr
				},
				SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error { return nil },
				RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
					return nil
				},
			}
			btxMock := &ClientIMock{
				GetTransactionBlocksFunc: func(ctx context.Context, transaction *blocktx_api.Transactions) (*blocktx_api.TransactionBlocks, error) {
					require.Equal(t, 3, len(transaction.GetTransactions()))

					return &blocktx_api.TransactionBlocks{TransactionBlocks: tc.blocks}, tc.getTransactionBlocksErr
				},
			}

			pm := p2p.NewPeerManagerMock()
			processor, err := NewProcessor(metamorphStore, pm, btxMock,
				WithProcessCheckIfMinedInterval(20*time.Millisecond),
				WithProcessExpiredTxsInterval(time.Hour),
			)
			require.NoError(t, err)
			defer processor.Shutdown()

			require.Equal(t, 0, processor.ProcessorResponseMap.Len())

			processor.ProcessorResponseMap.Set(testdata.TX1Hash, processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_STORED))
			processor.ProcessorResponseMap.Set(testdata.TX2Hash, processor_response.NewProcessorResponseWithStatus(testdata.TX2Hash, metamorph_api.Status_SEEN_ON_NETWORK))
			processor.ProcessorResponseMap.Set(testdata.TX3Hash, processor_response.NewProcessorResponseWithStatus(testdata.TX3Hash, metamorph_api.Status_REJECTED))
			processor.ProcessorResponseMap.Set(testdata.TX4Hash, processor_response.NewProcessorResponseWithStatus(testdata.TX4Hash, metamorph_api.Status_MINED))

			time.Sleep(25 * time.Millisecond)

			require.Equal(t, tc.expectedNrOfUpdates, len(metamorphStore.UpdateMinedCalls()))
			require.Equal(t, 1, len(btxMock.GetTransactionBlocksCalls()))
		})
	}
}

func TestProcessExpiredTransactions(t *testing.T) {
	tt := []struct {
		name    string
		retries uint32
	}{
		{
			name:    "expired txs - 0 retries",
			retries: 0,
		},
		{
			name:    "expired txs - 0 retries",
			retries: 16,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					return &store.StoreData{Hash: testdata.TX2Hash}, nil
				},
				SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error { return nil },
				RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
					return nil
				},
			}
			pm := p2p.NewPeerManagerMock()
			processor, err := NewProcessor(metamorphStore, pm, nil,
				WithProcessCheckIfMinedInterval(time.Hour),
				WithProcessExpiredTxsInterval(time.Millisecond*20),
				WithNow(func() time.Time {
					return time.Date(2033, 1, 1, 1, 0, 0, 0, time.UTC)
				}),
			)
			require.NoError(t, err)
			defer processor.Shutdown()

			require.Equal(t, 0, processor.ProcessorResponseMap.Len())

			respSent := processor_response.NewProcessorResponseWithStatus(testdata.TX1Hash, metamorph_api.Status_SENT_TO_NETWORK)
			respSent.Retries.Add(tc.retries)

			respAnnounced := processor_response.NewProcessorResponseWithStatus(testdata.TX2Hash, metamorph_api.Status_ANNOUNCED_TO_NETWORK)
			respAnnounced.Retries.Add(tc.retries)

			respAccepted := processor_response.NewProcessorResponseWithStatus(testdata.TX3Hash, metamorph_api.Status_ACCEPTED_BY_NETWORK)
			respAccepted.Retries.Add(tc.retries)

			processor.ProcessorResponseMap.Set(testdata.TX1Hash, respSent)
			processor.ProcessorResponseMap.Set(testdata.TX2Hash, respAnnounced)
			processor.ProcessorResponseMap.Set(testdata.TX3Hash, respAccepted)

			time.Sleep(50 * time.Millisecond)
		})
	}
}

func TestProcessorHealth(t *testing.T) {
	tt := []struct {
		name       string
		peersAdded int

		expectedErr error
	}{
		{
			name:       "3 healthy peers",
			peersAdded: 3,
		},
		{
			name:       "1 healthy peer",
			peersAdded: 1,

			expectedErr: ErrUnhealthy,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					return &store.StoreData{Hash: testdata.TX2Hash}, nil
				},
				SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error { return nil },
				RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
					return nil
				},
			}
			pm := p2p.NewPeerManagerMock()

			for i := 0; i < tc.peersAdded; i++ {
				err := pm.AddPeer(&p2p.PeerMock{})
				require.NoError(t, err)
			}

			processor, err := NewProcessor(metamorphStore, pm, nil,
				WithProcessCheckIfMinedInterval(time.Hour),
				WithProcessExpiredTxsInterval(time.Millisecond*20),
				WithNow(func() time.Time {
					return time.Date(2033, 1, 1, 1, 0, 0, 0, time.UTC)
				}),
			)
			require.NoError(t, err)
			defer processor.Shutdown()

			err = processor.Health()

			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
