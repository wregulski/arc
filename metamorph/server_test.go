package metamorph_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bitcoin-sv/arc/blocktx"
	"github.com/bitcoin-sv/arc/blocktx/blocktx_api"
	. "github.com/bitcoin-sv/arc/metamorph"
	"github.com/bitcoin-sv/arc/metamorph/metamorph_api"
	. "github.com/bitcoin-sv/arc/metamorph/mocks"
	"github.com/bitcoin-sv/arc/metamorph/processor_response"
	"github.com/bitcoin-sv/arc/metamorph/store"
	"github.com/bitcoin-sv/arc/metamorph/store/sqlite"
	"github.com/bitcoin-sv/arc/testdata"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"github.com/ordishs/go-utils/stat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate moq -pkg mocks -out ./mocks/processor_mock.go . ProcessorI
//go:generate moq -pkg mocks -out ./mocks/bitcon_mock.go . BitcoinNode

func setStoreTestData(t *testing.T, s store.MetamorphStore) {
	ctx := context.Background()
	err := s.Set(ctx, testdata.TX1Hash[:], &store.StoreData{
		StoredAt:      testdata.Time,
		AnnouncedAt:   testdata.Time.Add(1 * time.Second),
		MinedAt:       testdata.Time.Add(2 * time.Second),
		Hash:          testdata.TX1Hash,
		Status:        metamorph_api.Status_SENT_TO_NETWORK,
		CallbackUrl:   "https://test.com",
		CallbackToken: "token",
	})
	require.NoError(t, err)
	err = s.Set(ctx, testdata.TX2Hash[:], &store.StoreData{
		StoredAt:    testdata.Time,
		AnnouncedAt: testdata.Time.Add(1 * time.Second),
		MinedAt:     testdata.Time.Add(2 * time.Second),
		Hash:        testdata.TX2Hash,
		Status:      metamorph_api.Status_SEEN_ON_NETWORK,
	})
	require.NoError(t, err)
	err = s.Set(ctx, testdata.TX3Hash[:], &store.StoreData{
		StoredAt:    testdata.Time,
		AnnouncedAt: testdata.Time.Add(1 * time.Second),
		MinedAt:     testdata.Time.Add(2 * time.Second),
		Hash:        testdata.TX3Hash,
		Status:      metamorph_api.Status_MINED,
	})
	require.NoError(t, err)
	err = s.Set(ctx, testdata.TX4Hash[:], &store.StoreData{
		StoredAt:    testdata.Time,
		AnnouncedAt: testdata.Time.Add(1 * time.Second),
		MinedAt:     testdata.Time.Add(2 * time.Second),
		Hash:        testdata.TX4Hash,
		Status:      metamorph_api.Status_REJECTED,
	})
	require.NoError(t, err)
}

func TestNewServer(t *testing.T) {
	t.Run("NewServer", func(t *testing.T) {
		server := NewServer(nil, nil, nil)
		assert.IsType(t, &Server{}, server)
	})
}

func TestHealth(t *testing.T) {
	t.Run("Health", func(t *testing.T) {
		processor := &ProcessorIMock{}
		sentToNetworkStat := stat.NewAtomicStats()
		for i := 0; i < 10; i++ {
			sentToNetworkStat.AddDuration("test", 10*time.Millisecond)
		}
		expectedStats := &ProcessorStats{
			StartTime:      time.Now(),
			UptimeMillis:   "2000ms",
			QueueLength:    136,
			QueuedCount:    356,
			SentToNetwork:  sentToNetworkStat,
			ChannelMapSize: 22,
		}
		processor.GetStatsFunc = func(debugItems bool) *ProcessorStats {
			return expectedStats
		}
		processor.GetPeersFunc = func() ([]string, []string) {
			return []string{"peer1"}, []string{}
		}

		server := NewServer(nil, processor, nil)
		stats, err := server.Health(context.Background(), &emptypb.Empty{})
		assert.NoError(t, err)
		assert.Equal(t, expectedStats.ChannelMapSize, stats.GetMapSize())
		assert.Equal(t, expectedStats.QueuedCount, stats.GetQueued())
		assert.Equal(t, expectedStats.SentToNetwork.GetMap()["test"].GetCount(), stats.GetProcessed())
		assert.Equal(t, expectedStats.QueueLength, stats.GetWaiting())
		assert.Equal(t, float32(10), stats.GetAverage())
	})
}

func TestPutTransaction(t *testing.T) {
	t.Run("PutTransaction - ANNOUNCED", func(t *testing.T) {
		s, err := sqlite.New(true, "")
		require.NoError(t, err)

		processor := &ProcessorIMock{}

		client := &ClientIMock{}

		server := NewServer(s, processor, client)
		server.SetTimeout(100 * time.Millisecond)

		var txStatus *metamorph_api.TransactionStatus
		txRequest := &metamorph_api.TransactionRequest{
			RawTx: testdata.TX1RawBytes,
		}

		processor.ProcessTransactionFunc = func(ctx context.Context, req *ProcessorRequest) {
			time.Sleep(10 * time.Millisecond)

			req.ResponseChannel <- processor_response.StatusAndError{
				Hash:   testdata.TX1Hash,
				Status: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
			}
		}

		txStatus, err = server.PutTransaction(context.Background(), txRequest)
		assert.NoError(t, err)
		assert.Equal(t, metamorph_api.Status_ANNOUNCED_TO_NETWORK, txStatus.GetStatus())
		assert.True(t, txStatus.GetTimedOut())
	})

	t.Run("invalid request", func(t *testing.T) {
		server := NewServer(nil, nil, nil)

		txRequest := &metamorph_api.TransactionRequest{
			CallbackUrl: "api.callback.com",
		}

		_, err := server.PutTransaction(context.Background(), txRequest)
		assert.ErrorContains(t, err, "invalid URL [parse \"api.callback.com\": invalid URI for request]")
	})

	t.Run("PutTransaction - SEEN to network", func(t *testing.T) {
		s, err := sqlite.New(true, "")
		require.NoError(t, err)

		processor := &ProcessorIMock{}
		btc := &ClientIMock{}

		server := NewServer(s, processor, btc)

		var txStatus *metamorph_api.TransactionStatus
		txRequest := &metamorph_api.TransactionRequest{
			RawTx: testdata.TX1RawBytes,
		}

		processor.ProcessTransactionFunc = func(ctx context.Context, req *ProcessorRequest) {
			time.Sleep(10 * time.Millisecond)
			req.ResponseChannel <- processor_response.StatusAndError{
				Hash:   testdata.TX1Hash,
				Status: metamorph_api.Status_SEEN_ON_NETWORK,
			}
		}
		txStatus, err = server.PutTransaction(context.Background(), txRequest)
		assert.NoError(t, err)
		assert.Equal(t, metamorph_api.Status_SEEN_ON_NETWORK, txStatus.GetStatus())
		assert.False(t, txStatus.GetTimedOut())
	})

	t.Run("PutTransaction - Err", func(t *testing.T) {
		s, err := sqlite.New(true, "")
		require.NoError(t, err)

		processor := &ProcessorIMock{}
		btc := &ClientIMock{}

		server := NewServer(s, processor, btc)

		var txStatus *metamorph_api.TransactionStatus
		txRequest := &metamorph_api.TransactionRequest{
			RawTx:         testdata.TX1RawBytes,
			WaitForStatus: metamorph_api.Status_SENT_TO_NETWORK,
		}
		processor.ProcessTransactionFunc = func(ctx context.Context, req *ProcessorRequest) {
			time.Sleep(10 * time.Millisecond)
			req.ResponseChannel <- processor_response.StatusAndError{
				Hash:   testdata.TX1Hash,
				Status: metamorph_api.Status_REJECTED,
				Err:    fmt.Errorf("some error"),
			}
		}

		txStatus, err = server.PutTransaction(context.Background(), txRequest)
		assert.NoError(t, err)
		assert.Equal(t, metamorph_api.Status_REJECTED, txStatus.GetStatus())
		assert.Equal(t, "some error", txStatus.GetRejectReason())
		assert.False(t, txStatus.GetTimedOut())
	})
}

func TestServer_GetTransactionStatus(t *testing.T) {
	tests := []struct {
		name               string
		req                *metamorph_api.TransactionStatusRequest
		getTxMerklePathErr error
		getErr             error
		status             metamorph_api.Status
		merklePath         string

		want    *metamorph_api.TransactionStatus
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "GetTransactionStatus - not found",
			req: &metamorph_api.TransactionStatusRequest{
				Txid: "a147cc3c71cc13b29f18273cf50ffeb59fc9758152e2b33e21a8092f0b049118",
			},
			getErr: store.ErrNotFound,

			want: nil,
			wantErr: func(t assert.TestingT, err error, rest ...interface{}) bool {
				return assert.ErrorIs(t, err, store.ErrNotFound, rest...)
			},
		},
		{
			name: "GetTransactionStatus - test.TX1",
			req: &metamorph_api.TransactionStatusRequest{
				Txid: testdata.TX1,
			},
			status:     metamorph_api.Status_SENT_TO_NETWORK,
			merklePath: "00000",

			want: &metamorph_api.TransactionStatus{
				StoredAt:    timestamppb.New(testdata.Time),
				AnnouncedAt: timestamppb.New(testdata.Time.Add(1 * time.Second)),
				MinedAt:     timestamppb.New(testdata.Time.Add(2 * time.Second)),
				Txid:        testdata.TX1,
				Status:      metamorph_api.Status_SENT_TO_NETWORK,
				MerklePath:  "00000",
			},
			wantErr: assert.NoError,
		},
		{
			name: "GetTransactionStatus - test.TX1 - error",
			req: &metamorph_api.TransactionStatusRequest{
				Txid: testdata.TX1,
			},
			status:             metamorph_api.Status_SENT_TO_NETWORK,
			getTxMerklePathErr: errors.New("failed to get tx merkle path"),

			want: &metamorph_api.TransactionStatus{
				StoredAt:    timestamppb.New(testdata.Time),
				AnnouncedAt: timestamppb.New(testdata.Time.Add(1 * time.Second)),
				MinedAt:     timestamppb.New(testdata.Time.Add(2 * time.Second)),
				Txid:        testdata.TX1,
				Status:      metamorph_api.Status_SENT_TO_NETWORK,
				MerklePath:  "",
			},
			wantErr: assert.NoError,
		},
		{
			name: "GetTransactionStatus - test.TX1 - tx not found for Merkle path",
			req: &metamorph_api.TransactionStatusRequest{
				Txid: testdata.TX1,
			},
			status:             metamorph_api.Status_MINED,
			getTxMerklePathErr: blocktx.ErrTransactionNotFoundForMerklePath,

			want: &metamorph_api.TransactionStatus{
				StoredAt:    timestamppb.New(testdata.Time),
				AnnouncedAt: timestamppb.New(testdata.Time.Add(1 * time.Second)),
				MinedAt:     timestamppb.New(testdata.Time.Add(2 * time.Second)),
				Txid:        testdata.TX1,
				Status:      metamorph_api.Status_MINED,
				MerklePath:  "",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &ClientIMock{
				GetTransactionMerklePathFunc: func(ctx context.Context, transaction *blocktx_api.Transaction) (string, error) {
					return tt.merklePath, tt.getTxMerklePathErr
				},
			}

			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					data := &store.StoreData{
						StoredAt:      testdata.Time,
						AnnouncedAt:   testdata.Time.Add(1 * time.Second),
						MinedAt:       testdata.Time.Add(2 * time.Second),
						Hash:          testdata.TX1Hash,
						Status:        tt.status,
						CallbackUrl:   "https://test.com",
						CallbackToken: "token",
					}
					return data, tt.getErr
				},
				RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
					return nil
				},
			}

			server := NewServer(metamorphStore, nil, client)
			got, err := server.GetTransactionStatus(context.Background(), tt.req)
			if !tt.wantErr(t, err, fmt.Sprintf("GetTransactionStatus(%v)", tt.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetTransactionStatus(%v)", tt.req)
		})
	}
}

func TestValidateCallbackURL(t *testing.T) {
	tt := []struct {
		name        string
		callbackURL string

		expectedErrorStr string
	}{
		{
			name:        "empty callback URL",
			callbackURL: "",
		},
		{
			name:        "valid callback URL",
			callbackURL: "http://api.callback.com",
		},
		{
			name:        "invalid callback URL",
			callbackURL: "api.callback.com",

			expectedErrorStr: "invalid URL [parse \"api.callback.com\": invalid URI for request]",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCallbackURL(tc.callbackURL)

			if tc.expectedErrorStr != "" || err != nil {
				require.ErrorContains(t, err, tc.expectedErrorStr)
				return
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPutTransactions(t *testing.T) {
	hash0, err := chainhash.NewHashFromStr("9b58926ec7eed21ec2f3ca518d5fc0c6ccbf963e25c3e7ac496c99867d97599a")
	require.NoError(t, err)

	tx0, err := bt.NewTxFromString("010000000000000000ef016b51c656fb06639ea6c1c3642a5ede9ecf9f749b95cb47d4e57eda7a3953b1c64c0000006a47304402201ade53acd924e90c0aeabbf9085d075acb23c4712e7f728a23979a466ab55e19022047a85963ce2eddc21573b4a6c0e7ccfec44153e74f9d03d31f955ff486449240412102f87ce69f6ba5444aed49c34470041189c1e1060acd99341959c0594002c61bf0ffffffffe8030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac01e7030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac00000000")
	require.Equal(t, tx0.TxID(), hash0.String())

	require.NoError(t, err)
	tx1, err := bt.NewTxFromString("010000000000000000ef016b51c656fb06639ea6c1c3642a5ede9ecf9f749b95cb47d4e57eda7a3953b1c6660000006b483045022100e6d888a31cabb7bd491da63c9378d550ab728e6f81aa1c9420e1e055123e4728022040fd7263f08ecb53a1c9dbbc074d4b36e34e8db2ce78fed012a517052befda2b412102f87ce69f6ba5444aed49c34470041189c1e1060acd99341959c0594002c61bf0ffffffffe8030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac01e7030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac00000000")
	require.NoError(t, err)
	hash1, err := chainhash.NewHashFromStr("5d09daee7a648db6f99a7b678e9d64e6bf6867fb8a5f8818f4718b5a871fead1")
	require.NoError(t, err)
	require.Equal(t, tx1.TxID(), hash1.String())

	tx2, err := bt.NewTxFromString("010000000000000000ef016b51c656fb06639ea6c1c3642a5ede9ecf9f749b95cb47d4e57eda7a3953b1c6690000006a4730440220519b37c338888500e8299dd9afe462930352c95af1b436a29411b5eaaca7ec9c02204f821540a109323dbb36bd1d89bc057a435a4efbb5df7c3cae0d8522265cdd5c412102f87ce69f6ba5444aed49c34470041189c1e1060acd99341959c0594002c61bf0ffffffffe8030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac01e7030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac00000000")
	require.NoError(t, err)
	hash2, err := chainhash.NewHashFromStr("337bf4982dd12f399c1f20a7806c8005255355d8df84621062f572571f52f03b")
	require.NoError(t, err)
	require.Equal(t, tx2.TxID(), hash2.String())

	tt := []struct {
		name              string
		processorResponse map[string]*processor_response.StatusAndError
		transactionFound  map[int]*store.StoreData
		requests          *metamorph_api.TransactionRequests
		getErr            error

		expectedErrorStr                         string
		expectedStatuses                         *metamorph_api.TransactionStatuses
		expectedProcessorProcessTransactionCalls int
	}{
		{
			name: "single new transaction response seen on network - wait for sent to network status",
			requests: &metamorph_api.TransactionRequests{
				Transactions: []*metamorph_api.TransactionRequest{
					{
						RawTx:         tx0.Bytes(),
						WaitForStatus: metamorph_api.Status_SENT_TO_NETWORK,
					},
				},
			},
			processorResponse: map[string]*processor_response.StatusAndError{hash0.String(): {
				Hash:   hash0,
				Status: metamorph_api.Status_SEEN_ON_NETWORK,
				Err:    nil,
			}},

			expectedProcessorProcessTransactionCalls: 1,
			expectedStatuses: &metamorph_api.TransactionStatuses{
				Statuses: []*metamorph_api.TransactionStatus{
					{
						Txid:   hash0.String(),
						Status: metamorph_api.Status_SEEN_ON_NETWORK,
					},
				},
			},
		},
		{
			name: "single new transaction response with error",
			requests: &metamorph_api.TransactionRequests{
				Transactions: []*metamorph_api.TransactionRequest{
					{
						RawTx:         tx0.Bytes(),
						WaitForStatus: metamorph_api.Status_STORED,
					},
				},
			},
			processorResponse: map[string]*processor_response.StatusAndError{hash0.String(): {
				Hash:   hash0,
				Status: metamorph_api.Status_STORED,
				Err:    errors.New("unable to process transaction"),
			}},

			expectedProcessorProcessTransactionCalls: 1,
			expectedStatuses: &metamorph_api.TransactionStatuses{
				Statuses: []*metamorph_api.TransactionStatus{
					{
						Txid:         hash0.String(),
						Status:       metamorph_api.Status_STORED,
						RejectReason: "unable to process transaction",
					},
				},
			},
		},
		{
			name: "single new transaction no response",
			requests: &metamorph_api.TransactionRequests{
				Transactions: []*metamorph_api.TransactionRequest{{RawTx: tx0.Bytes()}},
			},

			expectedProcessorProcessTransactionCalls: 1,
			expectedStatuses: &metamorph_api.TransactionStatuses{
				Statuses: []*metamorph_api.TransactionStatus{
					{
						Txid:     hash0.String(),
						Status:   metamorph_api.Status_UNKNOWN,
						TimedOut: true,
					},
				},
			},
		},
		{
			name: "batch of 3 transactions",
			requests: &metamorph_api.TransactionRequests{
				Transactions: []*metamorph_api.TransactionRequest{
					{
						RawTx:         tx0.Bytes(),
						WaitForStatus: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					}, {
						RawTx:         tx1.Bytes(),
						WaitForStatus: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					}, {
						RawTx:         tx2.Bytes(),
						WaitForStatus: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					},
				},
			},
			transactionFound: map[int]*store.StoreData{1: {
				Status:      metamorph_api.Status_SENT_TO_NETWORK,
				Hash:        hash1,
				StoredAt:    time.Time{},
				AnnouncedAt: time.Time{},
				MinedAt:     time.Time{},
			}},
			processorResponse: map[string]*processor_response.StatusAndError{
				hash0.String(): {
					Hash:   hash0,
					Status: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					Err:    nil,
				},
				hash1.String(): {
					Hash:   hash1,
					Status: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					Err:    nil,
				},
				hash2.String(): {
					Hash:   hash2,
					Status: metamorph_api.Status_ACCEPTED_BY_NETWORK,
					Err:    nil,
				},
			},

			expectedProcessorProcessTransactionCalls: 3,
			expectedStatuses: &metamorph_api.TransactionStatuses{
				Statuses: []*metamorph_api.TransactionStatus{
					{
						Txid:   hash0.String(),
						Status: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					},
					{
						Txid:   hash1.String(),
						Status: metamorph_api.Status_ANNOUNCED_TO_NETWORK,
					},
					{
						Txid:   hash2.String(),
						Status: metamorph_api.Status_ACCEPTED_BY_NETWORK,
					},
				},
			},
		},
		{
			name: "failed to get tx",
			requests: &metamorph_api.TransactionRequests{
				Transactions: []*metamorph_api.TransactionRequest{
					{
						RawTx:         tx0.Bytes(),
						WaitForStatus: metamorph_api.Status_SENT_TO_NETWORK,
					},
				},
			},
			processorResponse: map[string]*processor_response.StatusAndError{hash0.String(): {
				Hash:   hash0,
				Status: metamorph_api.Status_SEEN_ON_NETWORK,
				Err:    nil,
			}},
			getErr: errors.New("failed to get tx"),

			expectedProcessorProcessTransactionCalls: 1,
			expectedStatuses: &metamorph_api.TransactionStatuses{
				Statuses: []*metamorph_api.TransactionStatus{
					{
						Txid:   hash0.String(),
						Status: metamorph_api.Status_SEEN_ON_NETWORK,
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			processor := &ProcessorIMock{
				ProcessTransactionFunc: func(_ context.Context, req *ProcessorRequest) {
					resp, found := tc.processorResponse[req.Data.Hash.String()]
					if found {
						req.ResponseChannel <- *resp
					}
				},
			}

			serverLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
			server := NewServer(nil, processor, nil, WithLogger(serverLogger))

			server.SetTimeout(5 * time.Second)
			statuses, err := server.PutTransactions(context.Background(), tc.requests)
			if tc.expectedErrorStr != "" || err != nil {
				require.ErrorContains(t, err, tc.expectedErrorStr)
				return
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expectedProcessorProcessTransactionCalls, len(processor.ProcessTransactionCalls()))

			for i := 0; i < len(tc.expectedStatuses.GetStatuses()); i++ {
				expected := tc.expectedStatuses.GetStatuses()[i]
				status := statuses.GetStatuses()[i]
				require.Equal(t, expected, status)
			}
		})
	}
}

func TestSetUnlockedbyName(t *testing.T) {
	tt := []struct {
		name            string
		recordsAffected int64
		errSetUnlocked  error

		expectedRecordsAffected int
		expectedErrorStr        string
	}{
		{
			name:            "success",
			recordsAffected: 5,

			expectedRecordsAffected: 5,
		},
		{
			name: "error",

			errSetUnlocked:   errors.New("failed to set unlocked"),
			expectedErrorStr: "failed to set unlocked",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					return &store.StoreData{}, nil
				},
				SetUnlockedByNameFunc: func(ctx context.Context, lockedBy string) (int64, error) {
					return tc.recordsAffected, tc.errSetUnlocked
				},
				RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
					return nil
				},
			}

			server := NewServer(metamorphStore, nil, nil)
			response, err := server.SetUnlockedByName(context.Background(), &metamorph_api.SetUnlockedByNameRequest{
				Name: "test",
			})

			if tc.expectedErrorStr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErrorStr)
				return
			}

			require.Equal(t, tc.expectedRecordsAffected, int(response.GetRecordsAffected()))
		})
	}
}

func TestStartGRPCServer(t *testing.T) {
	tt := []struct {
		name string
	}{
		{
			name: "start and shutdown",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			metamorphStore := &MetamorphStoreMock{
				GetFunc: func(ctx context.Context, key []byte) (*store.StoreData, error) {
					return &store.StoreData{}, nil
				},
				SetUnlockedFunc: func(ctx context.Context, hashes []*chainhash.Hash) error { return nil },
				RemoveCallbackerFunc: func(ctx context.Context, hash *chainhash.Hash) error {
					return nil
				},
			}

			btc := &ClientIMock{}

			processor := &ProcessorIMock{
				ShutdownFunc: func() {},
			}
			server := NewServer(metamorphStore, processor, btc)

			go func() {
				err := server.StartGRPCServer("localhost:7000", 10000)
				require.NoError(t, err)
			}()
			time.Sleep(50 * time.Millisecond)

			server.Shutdown()
		})
	}
}
