package dynamodb

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/bitcoin-sv/arc/metamorph/metamorph_api"
	"github.com/bitcoin-sv/arc/metamorph/store"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

const (
	hostname      = "test-host"
	host          = "http://localhost:"
	regionUsEast1 = "us-east-1"
	port          = "8000/tcp"
)

var (
	TX1Raw         = "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff1a0386c40b2f7461616c2e636f6d2f00cf47ad9c7af83836000000ffffffff0117564425000000001976a914522cf9e7626d9bd8729e5a1398ece40dad1b6a2f88ac00000000"
	TX1RawBytes, _ = hex.DecodeString(TX1Raw)
	TX1, _         = bt.NewTxFromBytes(TX1RawBytes)
	TX1Hash, _     = chainhash.NewHashFromStr(TX1.TxID())

	TX2raw         = "010000000000000000ef016f8828b2d3f8085561d0b4ff6f5d17c269206fa3d32bcd3b22e26ce659ed12e7000000006b483045022100d3649d120249a09af44b4673eecec873109a3e120b9610b78858087fb225c9b9022037f16999b7a4fecdd9f47ebdc44abd74567a18940c37e1481ab0fe84d62152e4412102f87ce69f6ba5444aed49c34470041189c1e1060acd99341959c0594002c61bf0ffffffffe7030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac01e7030000000000001976a914c2b6fd4319122b9b5156a2a0060d19864c24f49a88ac00000000"
	TX2RawBytes, _ = hex.DecodeString(TX2raw)
	TX2, _         = bt.NewTxFromBytes(TX2RawBytes)
	TX2Hash, _     = chainhash.NewHashFromStr(TX2.TxID())
	dateNow        = time.Date(2023, 11, 12, 13, 0, 0, 0, time.UTC)
)

func NewDynamoDBIntegrationTestRepo(t *testing.T) (*DynamoDB, *dynamodb.Client) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	resource, err := pool.Run("amazon/dynamodb-local", "latest", []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(t, err)
	})

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           host + resource.GetPort(port),
			SigningRegion: regionUsEast1,
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: "dummy", SecretAccessKey: "dummy", SessionToken: "dummy",
				Source: "Hard-coded credentials; values are irrelevant for local DynamoDB",
			},
		}),
	)
	require.NoError(t, err)

	client := dynamodb.NewFromConfig(cfg)

	pool.MaxWait = 60 * time.Second

	err = pool.Retry(func() error {
		_, err := client.ListTables(context.Background(), &dynamodb.ListTablesInput{})
		return err
	})
	require.NoError(t, err)

	repo, err := New(client, hostname, 1*time.Hour, "test-env", WithNow(func() time.Time { return dateNow }))
	require.NoError(t, err)

	tables, err := client.ListTables(context.Background(), &dynamodb.ListTablesInput{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"blocks-test-env", "transactions-test-env"}, tables.TableNames)

	return repo, client
}

func putItem(t *testing.T, ctx context.Context, client *dynamodb.Client, storeData any) {
	item, err := attributevalue.MarshalMap(storeData)
	require.NoError(t, err)
	// put item into table
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("transactions-test-env"), Item: item,
	})

	require.NoError(t, err)
}

func TestDynamoDBIntegration(t *testing.T) {
	Block1 := "0000000000000000072be13e375ffd673b1f37b0ec5ecde7b7e15b01f5685d07"
	Block1Hash, err := chainhash.NewHashFromStr(Block1)
	require.NoError(t, err)

	dataStatusSent := &store.StoreData{
		Hash:          TX1Hash,
		Status:        metamorph_api.Status_SENT_TO_NETWORK,
		CallbackUrl:   "http://callback.com",
		CallbackToken: "abcd",
		MerkleProof:   false,
		RawTx:         TX1RawBytes,
		LockedBy:      hostname,
	}

	repo, client := NewDynamoDBIntegrationTestRepo(t)
	ctx := context.Background()

	t.Run("set block processed", func(t *testing.T) {
		err := repo.SetBlockProcessed(ctx, Block1Hash)
		require.NoError(t, err)
	})

	t.Run("get block processed", func(t *testing.T) {
		processedAt, err := repo.GetBlockProcessed(ctx, Block1Hash)
		require.NoError(t, err)
		require.NotNil(t, processedAt)
	})

	t.Run("get - error", func(t *testing.T) {
		type invalid struct {
			Hash      *chainhash.Hash `dynamodbav:"tx_hash"`
			WrongType bool            `dynamodbav:"block_height"`
		}
		putItem(t, ctx, client, invalid{
			Hash:      TX1Hash,
			WrongType: false,
		})

		_, err := repo.Get(ctx, TX1Hash[:])

		_, isAttrErr := err.(*attributevalue.UnmarshalTypeError)
		require.True(t, isAttrErr)
	})
	t.Run("set/get", func(t *testing.T) {
		err := repo.Set(ctx, nil, dataStatusSent)
		require.NoError(t, err)

		returnedData, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, dataStatusSent, returnedData)
	})

	t.Run("set unlocked", func(t *testing.T) {
		err := repo.SetUnlocked(ctx, []*chainhash.Hash{TX1Hash})
		require.NoError(t, err)

		returnedData, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, lockedByNone, returnedData.LockedBy)

		dataStatusSent.LockedBy = lockedByNone
	})

	t.Run("get unmined", func(t *testing.T) {
		dataStatusAnnounced := &store.StoreData{
			Hash:     TX2Hash,
			Status:   metamorph_api.Status_ANNOUNCED_TO_NETWORK,
			RawTx:    TX2RawBytes,
			LockedBy: lockedByNone,
			StoredAt: dateNow,
		}
		putItem(t, ctx, client, dataStatusAnnounced)

		returnedData, err := repo.GetUnmined(ctx, time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC), 2)
		require.NoError(t, err)
		require.Contains(t, returnedData, dataStatusSent)
		require.Contains(t, returnedData, dataStatusAnnounced)

		tx1, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Contains(t, hostname, tx1.LockedBy)
		tx2, err := repo.Get(ctx, TX2Hash[:])
		require.NoError(t, err)
		require.Contains(t, hostname, tx2.LockedBy)
	})

	t.Run("set unlocked by name", func(t *testing.T) {
		results, err := repo.SetUnlockedByName(ctx, "this-does-not-exist")
		require.Equal(t, int64(0), results)
		require.NoError(t, err)

		results, err = repo.SetUnlockedByName(ctx, hostname)
		require.Equal(t, int64(2), results)
		require.NoError(t, err)

		returnedData, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, lockedByNone, returnedData.LockedBy)
		tx2, err := repo.Get(ctx, TX2Hash[:])
		require.NoError(t, err)
		require.Contains(t, lockedByNone, tx2.LockedBy)
	})

	t.Run("update status", func(t *testing.T) {
		err = repo.UpdateStatus(ctx, TX1Hash, metamorph_api.Status_REJECTED, "missing inputs")
		require.NoError(t, err)
		returnedDataRejected, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, metamorph_api.Status_REJECTED, returnedDataRejected.Status)
		require.Equal(t, "missing inputs", returnedDataRejected.RejectReason)
		require.Equal(t, TX1RawBytes, returnedDataRejected.RawTx)

		err = repo.UpdateStatus(ctx, TX1Hash, metamorph_api.Status_SEEN_IN_ORPHAN_MEMPOOL, "")
		require.NoError(t, err)
		returnedDataSeenInOrphanMempool, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, metamorph_api.Status_SEEN_IN_ORPHAN_MEMPOOL, returnedDataSeenInOrphanMempool.Status)
		require.Equal(t, TX1RawBytes, returnedDataSeenInOrphanMempool.RawTx)

		err = repo.UpdateStatus(ctx, TX1Hash, metamorph_api.Status_SEEN_ON_NETWORK, "")
		require.NoError(t, err)
		returnedDataSeenOnNetwork, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, metamorph_api.Status_SEEN_ON_NETWORK, returnedDataSeenOnNetwork.Status)
		require.Equal(t, TX1RawBytes, returnedDataSeenOnNetwork.RawTx)

		err = repo.UpdateStatus(ctx, TX1Hash, metamorph_api.Status_MINED, "")
		require.NoError(t, err)
		returnedDataMined, err := repo.Get(ctx, TX1Hash[:])
		require.NoError(t, err)
		require.Equal(t, metamorph_api.Status_MINED, returnedDataMined.Status)
		require.Equal(t, dateNow, returnedDataMined.MinedAt)
		require.Equal(t, TX1RawBytes, returnedDataMined.RawTx)
	})

	t.Run("update mined", func(t *testing.T) {
		err = repo.UpdateMined(ctx, TX2Hash, Block1Hash, 100)
		require.NoError(t, err)
		returnedData, err := repo.Get(ctx, TX2Hash[:])
		require.NoError(t, err)
		require.Equal(t, metamorph_api.Status_MINED, returnedData.Status)
		require.Equal(t, TX2RawBytes, returnedData.RawTx)
		require.Equal(t, dateNow, returnedData.MinedAt)
	})

	t.Run("del", func(t *testing.T) {
		err := repo.Del(ctx, TX1Hash[:])
		require.NoError(t, err)
		_, err = repo.Get(ctx, TX1Hash[:])
		require.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("blocks - time to live = -1 hour", func(t *testing.T) {
		repo.ttl = time.Minute * -10
		err := repo.SetBlockProcessed(ctx, Block1Hash)
		require.NoError(t, err)

		time.Sleep(2 * time.Second) // give DynamoDB time to delete
		processedAt, err := repo.GetBlockProcessed(ctx, Block1Hash)
		require.NoError(t, err)
		require.Nil(t, processedAt)
	})

	t.Run("transactions - time to live = -1 hour", func(t *testing.T) {
		repo.ttl = time.Minute * -10
		err := repo.Set(ctx, nil, dataStatusSent)
		require.NoError(t, err)

		time.Sleep(10 * time.Second) // give DynamoDB time to delete
		_, err = repo.Get(ctx, TX1Hash[:])
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}
