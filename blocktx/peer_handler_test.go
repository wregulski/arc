package blocktx

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/bitcoin-sv/arc/blocktx/blocktx_api"
	"github.com/bitcoin-sv/arc/blocktx/store"
	"github.com/libsv/go-bc"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-p2p"
	"github.com/libsv/go-p2p/bsvutil"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"github.com/libsv/go-p2p/wire"
	"github.com/ordishs/go-utils/expiringmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mocking wire.peerI as it's third party library and need to mock in here
//
//go:generate moq -out ./store/mock.go ./store Interface
type MockedPeer struct{}

func (peer *MockedPeer) Connected() bool                            { return true }
func (peer *MockedPeer) WriteMsg(msg wire.Message) error            { return nil }
func (peer *MockedPeer) String() string                             { return "" }
func (peer *MockedPeer) AnnounceTransaction(txHash *chainhash.Hash) {}
func (peer *MockedPeer) RequestTransaction(txHash *chainhash.Hash)  {}
func (peer *MockedPeer) AnnounceBlock(blockHash *chainhash.Hash)    {}
func (peer *MockedPeer) RequestBlock(blockHash *chainhash.Hash)     {}
func (peer *MockedPeer) Network() wire.BitcoinNet                   { return 0 }
func (peer *MockedPeer) IsHealthy() bool                            { return true }

func TestExtractHeight(t *testing.T) {
	coinbase, _ := hex.DecodeString("01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff570350cc0b041547b5630cfabe6d6d0000000000000000000000000000000000000000000000000000000000000000010000000000000047ed20542096bd0000000000143362663865373833636662643732306431383436000000000140be4025000000001976a914c9b0abe09b7dd8e9d1e8c1e3502d32ab0d7119e488ac00000000")
	tx, err := bsvutil.NewTxFromBytes(coinbase)
	require.NoError(t, err)

	buff := bytes.NewBuffer(nil)
	err = tx.MsgTx().Serialize(buff)
	require.NoError(t, err)
	btTx, err := bt.NewTxFromBytes(buff.Bytes())
	require.NoError(t, err)

	height := extractHeightFromCoinbaseTx(btTx)

	assert.Equalf(t, uint64(773200), height, "height should be 773200, got %d", height)
}

func TestExtractHeightForRegtest(t *testing.T) {
	coinbase, _ := hex.DecodeString("02000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0502dc070101ffffffff012f500900000000002321032efe256e14fd77eea05d0453374f8920e0a7a4a573bb3937ef3f567f3937129cac00000000")
	tx, err := bsvutil.NewTxFromBytes(coinbase)
	require.NoError(t, err)

	buff := bytes.NewBuffer(nil)
	err = tx.MsgTx().Serialize(buff)
	require.NoError(t, err)
	btTx, err := bt.NewTxFromBytes(buff.Bytes())
	require.NoError(t, err)

	height := extractHeightFromCoinbaseTx(btTx)

	assert.Equalf(t, uint64(2012), height, "height should be 2012, got %d", height)
}

func TestGetAnnouncedCacheBlockHashes(t *testing.T) {
	peerHandler := PeerHandler{
		announcedCache: expiringmap.New[chainhash.Hash, []p2p.PeerI](5 * time.Minute),
	}

	peer, err := p2p.NewPeerMock("", &peerHandler, wire.MainNet)
	assert.NoError(t, err)

	hash, err := chainhash.NewHashFromStr("00000000000000000e3c9aafb4c823562dd38f15b75849be348131a785154e33")
	assert.NoError(t, err)
	peerHandler.announcedCache.Set(*hash, []p2p.PeerI{peer})

	hash, err = chainhash.NewHashFromStr("00000000000000000cd097bf90c0f8480b930c88f3994503abccf45d579c601c")
	assert.NoError(t, err)
	peerHandler.announcedCache.Set(*hash, []p2p.PeerI{peer})

	hashes := peerHandler.getAnnouncedCacheBlockHashes()

	assert.ElementsMatch(t, hashes, []string{"00000000000000000e3c9aafb4c823562dd38f15b75849be348131a785154e33", "00000000000000000cd097bf90c0f8480b930c88f3994503abccf45d579c601c"})
}

func TestHandleBlock(t *testing.T) {
	// define HandleBlock function parameters (BlockMessage and p2p.PeerI)

	prevBlockHash1573650, _ := chainhash.NewHashFromStr("00000000000007b1f872a8abe664223d65acd22a500b1b8eb5db3fe09a9837ff")
	merkleRootHash1573650, _ := chainhash.NewHashFromStr("3d64b2bb6bd4e85aacb6d1965a2407fa21846c08dd9a8616866ad2f5c80fda7f")

	prevBlockHash1584899, _ := chainhash.NewHashFromStr("000000000000370a7d710d5d24968567618fa0c707950890ba138861fb7c9879")
	merkleRootHash1584899, _ := chainhash.NewHashFromStr("de877b5f2ef9f3e294ce44141c832b84efabea0d825fd3aa7024f23c38feb696")

	prevBlockHash1585018, _ := chainhash.NewHashFromStr("00000000000003fe2dc7e6ca0a37cb36a00742459e65a048d5bee0fc33d9ad32")
	merkleRootHash1585018, _ := chainhash.NewHashFromStr("9c1fe95a7ac4502e281f4f2eaa2902e12b0f486cf610977c73afb3cd060bebde")

	tt := []struct {
		name          string
		prevBlockHash chainhash.Hash
		merkleRoot    chainhash.Hash
		height        uint64
		txHashes      []string
		size          uint64
		nonce         uint32
		getBlockErr   error
	}{
		{
			name:          "block height 1573650",
			txHashes:      []string{"3d64b2bb6bd4e85aacb6d1965a2407fa21846c08dd9a8616866ad2f5c80fda7f"},
			prevBlockHash: *prevBlockHash1573650,
			merkleRoot:    *merkleRootHash1573650,
			height:        1573650,
			nonce:         3694498168,
			size:          216,
		},
		{
			name: "block height 1584899",
			txHashes: []string{
				"30f00edf09d7c4483509a52962e2e6ddfd16a0a146b9068288b1a5a2242e5c7b",
				"63dc4a8c11ec26e141f501e5c0dfa19b463eb5660e483ca5e0c8520979bb37bb",
				"fe220040445774788309ef0399939b70b90f7182dbf3ff24b2eaf6eeac04d395",
				"dcd51904bc0e58199b0c6fa37b8fe3b6f8ba696e6af8ecff27fe181f173346f4",
				"192ec6b58f1087f68728aabac2ce37ebe66e9bfc6f3af51cd39a2535e1100353",
				"e45955e1b4b7d184ffa3f2469f18b4f9b604dce1ba2265523ec2f407ed99ee14",
				"1d03c4f081a9c41b6ec1e45c1edb411de2765f0df3c7dfd5c91f49509af18960",
				"7607fabbd665e1b540647d0df197ec272751257a83265fe6d312909909c25827",
				"4c870f373eac5fb6f0a9e98dce2970047ad9c9f5b0479ae78bab86432439718a",
				"0e28a91a0ff248ef33dba449299a6663b5401f32695b22cb5ee21e0cd2a822d9",
				"d7f5f4ba7d1ae16cc6ff320693bc4299b4117e64afb0e2cc0634950d5a4d054f",
				"c4cebb360bc82d1a6bd1aad631a825ec0dd57eea6964b29551616486255399e1",
				"6346a7249eb0c40efcd5674f0f022e17b720d6f263be2cd2637326f3ee80d16f",
				"d0d4eaaf40a4414f11f895b66ee0ecbe2f71033b45e2faeea2805c9c1da976ef",
			},
			prevBlockHash: *prevBlockHash1584899,
			merkleRoot:    *merkleRootHash1584899,
			height:        1584899,
			nonce:         1234660301,
			size:          3150,
		},
		{
			name: "block height 1585018",
			txHashes: []string{
				"be181e91217d5f802f695e52144078f8dfbe51b8a815c3d6fb48c0d853ec683b",
				"354cb5b9b3586cca8b82025e7a08f1532fc51128d12c0bbf683f54dbb228efac",
				"2c64a04825dfcc0b87d9f31756d590530bf8c12ccf6670275d4970fb954f50e8",
				"cf5211f97fd59250967a17a0ec865665c9232b0b6ee2faa1e13462161a5509eb",
				"f53c84e09a1628eced8160b14b725cb184d46bf4ee92688372ef019f484ed214",
				"2a5d9ab4e810280dd994dc2eaf7fbe17b245e79b7808297b96f1e0dcc1b37dc6",
				"bdd25de67ea06af3651650a991dc742b4a56ee0707a498fc3aade4343a87fe6d",
				"7d4c950c903f8e4f027bbc5ef34ed189ace85e97ae938591cd5f35b6d2c81dfc",
				"13f2cc98b6a0dd7868853fb7d062391bd0f5c7fe759cf5dc25e269967b36c758",
				"2c9847577ca9ad986b3d1698c03c138d7160c50c16df36bbed2904c1d0b081a0",
				"5abeb598521c1c882f53543fc76bd2321f8bc154b25bccb177dc42f7879a66d8",
				"698a2a78ec1df92355878d9b94cca0a3a15008d15896e24697d5f2c3fb4f0b4b",
				"705a39d2accb41396023a58efbb07e7d508441d21abb0eb9c86a0f7070d4c697",
				"911de4d920159eb622be70f4323c572fe9dd5296e0e2be0611c04920234b810c",
				"b672c3a3a36ce458c2f9424bf35f30fa901580ea07483952d87cedac1c1cb9c0",
				"55b42c74269fd4e38ed1af18793a6e4cba9bba98ab07f7557ab7a05e03f9c74e",
				"89bf69fb351780acf4355a724bcb235374fd9be9fa5872686344896564831989",
				"32a852381b173ca5a2e1119c915c1c0b86df05e6a6198a857b47b098ab5181e1",
				"c79f4a9ff600f50f8da1f876d61aabfd13528d42ddc7f287eee87463439037ca",
				"ce6edcf9908746dac19ac6930d28e1709194d07175da22e9cfe60c54a4ad5f68",
				"c81f1b73a08c471a0bb2ba89fb187511ac35ccc074ca83298a84af1415e84102",
				"0c70102c0d4b87b39a81224981b5a45efdd72cc58c68aa63fd2f055cf5a17cf1",
				"c2038799cb0cab540a9ccd341b2e668ab59a495464b34b277933392f2dba8c12",
				"b33853a4512d22b65cc2ba188e852ca76dd094ab648f461c35875e13b2bb6562",
				"394450c8925e85950334dc1bbbb6387ba82d11337243fff6f8cfde80c7fc076c",
				"bab71c7a6b3d28714f41459dea64ee81ceb0f595e58623b182ab7fd1cf52201e",
				"815b4ec9bec4704c2cf18f0c40a63506a50a647ed5cd4ac7d5c07b0e0e474b0e",
				"3e28c488913e02a47ca81a58e60e6b26c7b483a83d04e7ef6af40ce4cb0fe016",
				"7b5b21da0cedf04bcde86da5dd6bc0db94939053f41d62eb85c950f4fff438e9",
				"1acad7f15e2d4e293949312db3b38d850e7ecb474615ba2df5daad9a0b375e63",
				"a010a3f4171eb88b43d1bc7bbff5a60a5422fb09e8d7e5bc74419b497fa9c63c",
				"df2f423e062e662fad0b29dbd284def3316707767da745f37d1ee5b6beb25781",
				"986a1ff920690de8819abc3d16c91e3cae30d3c5524a6bf895bbb54890736754",
				"056252b2fc1c5b49b7960c27b41b7dfcc28f4087d6c1095ae9dff040d8a39152",
				"1c4708b13d5d1600b1b3b02f5e224aa1edb6730a26ecb0a46e3668793bb4d52b",
				"d97f66206650361ba9bc975266d086692a25c31024f9f4ceaa6e43367787f941",
				"3664bb8205fa806515f1a128ae3c760e34fcb1e78d30c00d2b0bf3ea7d833ef7",
				"d5243d3bc735898c7284b3c7c368fc06543b129c2f2b40836d55f0be5107bf85",
				"e1b74f95639dbe35d01fd72b27214beb224e601c93c669220988f295d948a985",
				"0d93038b34f1d024ee7d942f253b8218d38a3f19e580ec6d7700b24801b62cb2",
			},
			prevBlockHash: *prevBlockHash1585018,
			merkleRoot:    *merkleRootHash1585018,
			height:        1584899,
			nonce:         1428255133,
			size:          8191650,
		},
		{
			name:          "get block error",
			txHashes:      []string{"3d64b2bb6bd4e85aacb6d1965a2407fa21846c08dd9a8616866ad2f5c80fda7f"},
			prevBlockHash: *prevBlockHash1573650,
			merkleRoot:    *merkleRootHash1573650,
			height:        1573650,
			nonce:         3694498168,
			size:          216,
			getBlockErr:   errors.New("failed to get block"),
		},
		{
			name:          "get block error - not found",
			txHashes:      []string{"3d64b2bb6bd4e85aacb6d1965a2407fa21846c08dd9a8616866ad2f5c80fda7f"},
			prevBlockHash: *prevBlockHash1573650,
			merkleRoot:    *merkleRootHash1573650,
			height:        1573650,
			nonce:         3694498168,
			size:          216,
			getBlockErr:   store.ErrBlockNotFound,
		},
	}

	for _, tc := range tt {
		batchSize := 4
		storeMock := &store.InterfaceMock{
			GetBlockFunc: func(ctx context.Context, hash *chainhash.Hash) (*blocktx_api.Block, error) {
				return &blocktx_api.Block{}, tc.getBlockErr
			},
			InsertBlockFunc: func(ctx context.Context, block *blocktx_api.Block) (uint64, error) {
				return 0, nil
			},
			MarkBlockAsDoneFunc: func(ctx context.Context, hash *chainhash.Hash, size uint64, txCount uint64) error {
				return nil
			},
			GetPrimaryFunc: func(ctx context.Context) (string, error) {
				hostName, err := os.Hostname()
				return hostName, err
			},
			TryToBecomePrimaryFunc: func(ctx context.Context, myHostName string) error {
				return nil
			},

			// main assert for the test to make sure block with a single transaction doesn't have any merkle paths other than empty ones "0000"
			UpdateBlockTransactionsFunc: func(ctx context.Context, blockId uint64, transactions []*blocktx_api.TransactionAndSource, merklePaths []string) error {
				assert.Equal(t, uint64(1), uint64(len(merklePaths)))
				assert.Equal(t, merklePaths[0], "fe12031800010100027fda0fc8f5d26a8616869add086c8421fa07245a96d1b6ac5ae8d46bbbb2643d")
				return nil
			},
		}

		// build peer manager
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		peerHandler, err := NewPeerHandler(logger, storeMock, 100, []string{}, wire.TestNet, WithTransactionBatchSize(batchSize))
		require.NoError(t, err)

		t.Run(tc.name, func(t *testing.T) {
			expectedInsertedTransactions := []*blocktx_api.TransactionAndSource{}
			transactionHashes := make([]*chainhash.Hash, len(tc.txHashes))
			for i, hash := range tc.txHashes {
				txHash, err := chainhash.NewHashFromStr(hash)
				require.NoError(t, err)
				transactionHashes[i] = txHash

				expectedInsertedTransactions = append(expectedInsertedTransactions, &blocktx_api.TransactionAndSource{Hash: txHash[:]})
			}

			var insertedBlockTransactions []*blocktx_api.TransactionAndSource

			storeMock.UpdateBlockTransactionsFunc = func(ctx context.Context, blockId uint64, transactions []*blocktx_api.TransactionAndSource, merklePaths []string) error {
				require.True(t, len(merklePaths) <= batchSize)
				require.True(t, len(transactions) <= batchSize)

				for i, path := range merklePaths {
					bump, err := bc.NewBUMPFromStr(path)
					require.NoError(t, err)
					tx, err := chainhash.NewHash(transactions[i].GetHash())
					require.NoError(t, err)
					root, err := bump.CalculateRootGivenTxid(tx.String())
					require.NoError(t, err)

					require.Equal(t, root, tc.merkleRoot.String())
				}

				insertedBlockTransactions = append(insertedBlockTransactions, transactions...)
				return nil
			}

			peer := &MockedPeer{}

			blockMessage := &p2p.BlockMessage{
				Header: &wire.BlockHeader{
					Version:    541065216,
					PrevBlock:  tc.prevBlockHash,
					MerkleRoot: tc.merkleRoot,
					Bits:       436732028,
					Nonce:      tc.nonce,
				},
				Height:            tc.height,
				TransactionHashes: transactionHashes,
				Size:              tc.size,
			}

			// call tested function
			err := peerHandler.HandleBlock(blockMessage, peer)
			require.NoError(t, err)

			require.ElementsMatch(t, expectedInsertedTransactions, insertedBlockTransactions)
			peerHandler.Shutdown()
		})
	}
}

func TestFillGaps(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)
	hash822014, err := chainhash.NewHashFromStr("0000000000000000025855b62f4c2e3732dad363a6f2ead94e4657ef96877067")
	require.NoError(t, err)
	hash822019, err := chainhash.NewHashFromStr("00000000000000000364332e1bbd61dc928141b9469c5daea26a4b506efc9656")
	require.NoError(t, err)
	tt := []struct {
		name              string
		blockGaps         []*store.BlockGap
		getBlockGapsErr   error
		primaryBlocktxErr error
		hostname          string

		expectedGetBlockGapsCalls int
		expectedErrorStr          string
	}{
		{
			name:      "success - no gaps",
			blockGaps: []*store.BlockGap{},
			hostname:  hostname,

			expectedGetBlockGapsCalls: 1,
		},
		{
			name: "success - 2 gaps",
			blockGaps: []*store.BlockGap{
				{
					Height: 822014,
					Hash:   hash822014,
				},
				{
					Height: 8220119,
					Hash:   hash822019,
				},
			},
			hostname: hostname,

			expectedGetBlockGapsCalls: 1,
		},
		{
			name:            "error getting block gaps",
			blockGaps:       []*store.BlockGap{},
			getBlockGapsErr: errors.New("failed to get block gaps"),
			hostname:        hostname,

			expectedGetBlockGapsCalls: 1,
			expectedErrorStr:          "failed to get block gaps",
		},
		{
			name:      "not primary",
			blockGaps: []*store.BlockGap{},
			hostname:  "not primary",

			expectedGetBlockGapsCalls: 0,
		},
		{
			name:              "check primary - error",
			blockGaps:         []*store.BlockGap{},
			hostname:          "not primary",
			primaryBlocktxErr: errors.New("failed to check primary"),

			expectedGetBlockGapsCalls: 0,
			expectedErrorStr:          "failed to check primary",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			const batchSize = 4

			var storeMock = &store.InterfaceMock{
				GetBlockGapsFunc: func(ctx context.Context, heightRange int) ([]*store.BlockGap, error) {
					return tc.blockGaps, tc.getBlockGapsErr
				},
				GetPrimaryFunc: func(ctx context.Context) (string, error) {
					return tc.hostname, tc.primaryBlocktxErr
				},
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
			peerHandler, err := NewPeerHandler(logger, storeMock, 100, []string{}, wire.TestNet, WithTransactionBatchSize(batchSize))
			require.NoError(t, err)
			peer := &MockedPeer{}
			err = peerHandler.FillGaps(peer)

			require.Equal(t, tc.expectedGetBlockGapsCalls, len(storeMock.GetBlockGapsCalls()))
			peerHandler.Shutdown()
			if tc.expectedErrorStr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErrorStr)
				return
			}
		})
	}
}

func TestStartFillGaps(t *testing.T) {
	hash822014, err := chainhash.NewHashFromStr("0000000000000000025855b62f4c2e3732dad363a6f2ead94e4657ef96877067")
	require.NoError(t, err)
	hostname, err := os.Hostname()
	require.NoError(t, err)

	tt := []struct {
		name            string
		hostname        string
		getBlockGapsErr error
	}{
		{
			name:     "success",
			hostname: hostname,
		},
		{
			name:            "error getting block gaps",
			hostname:        hostname,
			getBlockGapsErr: errors.New("failed to get block gaps"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			storeMock := &store.InterfaceMock{
				GetBlockGapsFunc: func(ctx context.Context, heightRange int) ([]*store.BlockGap, error) {
					return []*store.BlockGap{
						{
							Height: 822014,
							Hash:   hash822014,
						},
					}, tc.getBlockGapsErr
				},
				GetPrimaryFunc: func(ctx context.Context) (string, error) {
					return tc.hostname, nil
				},
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
			peerHandler, err := NewPeerHandler(logger, storeMock, 100, []string{"127.0.0.1:18333", "127.0.0.2:18333", "127.0.0.3:18333"}, wire.TestNet, WithFillGapsInterval(time.Millisecond*20))
			require.NoError(t, err)

			time.Sleep(120 * time.Millisecond)
			peerHandler.Shutdown()
		})
	}
}
