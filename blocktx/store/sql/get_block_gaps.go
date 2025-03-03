package sql

import (
	"context"

	"github.com/bitcoin-sv/arc/blocktx/store"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"github.com/ordishs/gocore"
)

func (s *SQL) GetBlockGaps(ctx context.Context, blockHeightRange int) ([]*store.BlockGap, error) {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("blocktx").NewStat("GetBlockGaps").AddTime(start)
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := `SELECT missing_blocks.missing_block_height, prevhash AS hash FROM blocks
				JOIN (
				SELECT bl.block_heights AS missing_block_height FROM (
				SELECT unnest(ARRAY(
					SELECT a.n
					FROM generate_series((SELECT max(height) - $1 AS block_height FROM blocks b), (SELECT max(height) AS block_height FROM blocks b)) AS a(n)
				)) AS block_heights) AS bl
				LEFT JOIN blocks blks ON blks.height = bl.block_heights
				WHERE blks.height IS NULL
				) AS missing_blocks ON blocks.height = missing_blocks.missing_block_height + 1
				ORDER BY missing_blocks.missing_block_height DESC;`

	rows, err := s.db.QueryContext(ctx, q, blockHeightRange)
	if err != nil {
		return nil, err
	}
	blockGaps := make([]*store.BlockGap, 0)
	for rows.Next() {
		var height uint64
		var hash []byte
		err = rows.Scan(&height, &hash)
		if err != nil {
			return nil, err
		}

		txHash, err := chainhash.NewHash(hash)
		if err != nil {
			return nil, err
		}

		blockGaps = append(blockGaps, &store.BlockGap{Height: height, Hash: txHash})
	}

	return blockGaps, nil
}
