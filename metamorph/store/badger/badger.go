package badger

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bitcoin-sv/arc/metamorph/metamorph_api"
	"github.com/bitcoin-sv/arc/metamorph/store"
	"github.com/dgraph-io/badger/v3"
	"github.com/libsv/go-p2p/chaincfg/chainhash"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/ordishs/go-utils"
	"github.com/ordishs/gocore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/spf13/viper"
)

func init() {
	badgerExpvarCollector := collectors.NewExpvarCollector(map[string]*prometheus.Desc{
		"badger_blocked_puts_total":   prometheus.NewDesc("badger_blocked_puts_total", "Blocked Puts", nil, nil),
		"badger_disk_reads_total":     prometheus.NewDesc("badger_disk_reads_total", "Disk Reads", nil, nil),
		"badger_disk_writes_total":    prometheus.NewDesc("badger_disk_writes_total", "Disk Writes", nil, nil),
		"badger_gets_total":           prometheus.NewDesc("badger_gets_total", "Gets", nil, nil),
		"badger_puts_total":           prometheus.NewDesc("badger_puts_total", "Puts", nil, nil),
		"badger_memtable_gets_total":  prometheus.NewDesc("badger_memtable_gets_total", "Memtable gets", nil, nil),
		"badger_lsm_size_bytes":       prometheus.NewDesc("badger_lsm_size_bytes", "LSM Size in bytes", []string{"database"}, nil),
		"badger_vlog_size_bytes":      prometheus.NewDesc("badger_vlog_size_bytes", "Value Log Size in bytes", []string{"database"}, nil),
		"badger_pending_writes_total": prometheus.NewDesc("badger_pending_writes_total", "Pending Writes", []string{"database"}, nil),
		"badger_read_bytes":           prometheus.NewDesc("badger_read_bytes", "Read bytes", nil, nil),
		"badger_written_bytes":        prometheus.NewDesc("badger_written_bytes", "Written bytes", nil, nil),
		"badger_lsm_bloom_hits_total": prometheus.NewDesc("badger_lsm_bloom_hits_total", "LSM Bloom Hits", []string{"level"}, nil),
		"badger_lsm_level_gets_total": prometheus.NewDesc("badger_lsm_level_gets_total", "LSM Level Gets", []string{"level"}, nil),
	})
	prometheus.MustRegister(badgerExpvarCollector)
}

type Badger struct {
	store  *badger.DB
	logger utils.Logger
	mu     sync.RWMutex
}

type loggerWrapper struct {
	*gocore.Logger
}

func (l loggerWrapper) Warningf(format string, args ...interface{}) {
	l.Warnf(format, args...)
}

func New(dir string) (*Badger, error) {
	logLevel := viper.GetString("logLevel")
	logger := loggerWrapper{gocore.Log("bdgr", gocore.NewLogLevelFromString(logLevel))}

	opts := badger.DefaultOptions(dir).
		WithLogger(logger).
		WithLoggingLevel(badger.ERROR).WithNumMemtables(32).
		WithMetricsEnabled(true)
	s, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	badgerStore := &Badger{
		store:  s,
		logger: logger,
	}

	return badgerStore, nil
}

func (s *Badger) SetUnlocked(ctx context.Context, hashes []*chainhash.Hash) error {
	return nil
}

func (s *Badger) SetUnlockedByName(ctx context.Context, lockedBy string) (int64, error) {
	return 0, nil
}

func (s *Badger) Close(ctx context.Context) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("Close").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:Close")
	defer span.Finish()

	metrics := s.store.BlockCacheMetrics()
	fmt.Printf("metrics: %+v", metrics)

	metrics2 := s.store.IndexCacheMetrics()
	fmt.Printf("metrics2: %+v", metrics2)

	return s.store.Close()
}

func (s *Badger) Set(ctx context.Context, key []byte, value *store.StoreData) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("Set").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:Set")
	defer span.Finish()

	if value.StoredAt.IsZero() {
		value.StoredAt = time.Now()
	}

	data, err := value.EncodeToBytes()
	if err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to encode data: %w", err)
	}

	if err = s.store.Update(func(tx *badger.Txn) error {
		return tx.Set(key, data)
	}); err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to set data: %w", err)
	}

	return nil
}

func (s *Badger) Get(ctx context.Context, hash []byte) (*store.StoreData, error) {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("Get").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:Get")
	defer span.Finish()

	var result *store.StoreData

	err := s.store.View(func(tx *badger.Txn) error {
		data, err := tx.Get(hash)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return store.ErrNotFound
			}
			span.SetTag(string(ext.Error), true)
			span.LogFields(log.Error(err))
			return err
		}

		if err = data.Value(func(val []byte) error {
			if result, err = store.DecodeFromBytes(val); err != nil {
				return err
			}
			return nil
		}); err != nil {
			span.SetTag(string(ext.Error), true)
			span.LogFields(log.Error(err))
			return fmt.Errorf("failed to decode data: %w", err)
		}

		return nil
	})

	return result, err
}

// UpdateStatus attempts to update the status of a transaction
func (s *Badger) UpdateStatus(ctx context.Context, hash *chainhash.Hash, status metamorph_api.Status, rejectReason string) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("UpdateStatus").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:UpdateStatus")
	defer span.Finish()

	// we need a lock here since we are doing 2 operations that need to be atomic
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.Get(ctx, hash[:])
	if err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return err
	}

	tx.Status = status
	tx.RejectReason = rejectReason

	// set the time the transaction was announced to the network
	switch status {
	case metamorph_api.Status_ANNOUNCED_TO_NETWORK:
		tx.AnnouncedAt = time.Now()
	case metamorph_api.Status_MINED:
		tx.MinedAt = time.Now()
	}

	if err = s.Set(ctx, hash[:], tx); err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to update data: %w", err)
	}

	return nil
}

func (s *Badger) RemoveCallbacker(ctx context.Context, hash *chainhash.Hash) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("RemoveCallbacker").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:RemoveCallbacker")
	defer span.Finish()

	// we need a lock here since we are doing 2 operations that need to be atomic
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.Get(ctx, hash[:])
	if err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return err
	}

	tx.CallbackUrl = ""

	if err = s.Set(ctx, hash[:], tx); err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to update data: %w", err)
	}

	return nil
}

// UpdateMined updates the transaction to mined
func (s *Badger) UpdateMined(ctx context.Context, hash *chainhash.Hash, blockHash *chainhash.Hash, blockHeight uint64) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("UpdateMined").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:UpdateMined")
	defer span.Finish()

	// we need a lock here since we are doing 2 operations that need to be atomic
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.Get(ctx, hash[:])
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// no need to update status if we don't have the transaction
			// we also shouldn't need to return an error here
			return nil
		}
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return err
	}

	tx.Status = metamorph_api.Status_MINED
	tx.BlockHash = blockHash
	tx.BlockHeight = blockHeight
	if err = s.Set(ctx, hash[:], tx); err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to update data: %w", err)
	}

	return nil
}

// GetUnmined returns all transactions that have not been mined
func (s *Badger) GetUnmined(ctx context.Context, since time.Time, limit int64) ([]*store.StoreData, error) {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("GetUnmined").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:GetUnmined")

	defer span.Finish()
	data := make([]*store.StoreData, 0)
	err := s.store.View(func(tx *badger.Txn) error {
		iter := tx.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			if strings.HasPrefix(string(item.Key()), "block_processed_") {
				continue
			}
			if item.IsDeletedOrExpired() {
				continue
			}

			var result *store.StoreData
			err := item.Value(func(val []byte) error {
				var err2 error
				if result, err2 = store.DecodeFromBytes(val); err2 != nil {
					return err2
				}
				return nil
			})
			if err != nil {
				span.SetTag(string(ext.Error), true)
				span.LogFields(log.Error(err))
				s.logger.Errorf("failed to decode data for %s: %w", item.Key(), err)
				continue
			}

			if result.StoredAt.Before(since) {
				continue
			}

			if result.Status < metamorph_api.Status_MINED || result.Status == metamorph_api.Status_SEEN_IN_ORPHAN_MEMPOOL {
				data = append(data, result)
			}

			if int64(len(data)) >= limit {
				return nil
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Badger) Del(ctx context.Context, hash []byte) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("Del").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:Del")
	defer span.Finish()

	return s.store.Update(func(tx *badger.Txn) error {
		return tx.Delete(hash)
	})
}

func (s *Badger) GetBlockProcessed(ctx context.Context, blockHash *chainhash.Hash) (*time.Time, error) {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("GetBlockProcessed").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:GetBlockProcessed")
	defer span.Finish()

	var result *time.Time

	key := append([]byte("block_processed_"), blockHash[:]...)

	err := s.store.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			span.SetTag(string(ext.Error), true)
			span.LogFields(log.Error(err))
			return err
		}

		if err = item.Value(func(val []byte) error {
			dec := gob.NewDecoder(bytes.NewReader(val))
			return dec.Decode(&result)
		}); err != nil {
			span.SetTag(string(ext.Error), true)
			span.LogFields(log.Error(err))
			return fmt.Errorf("failed to decode data: %w", err)
		}

		return nil
	})

	return result, err
}

func (s *Badger) SetBlockProcessed(ctx context.Context, blockHash *chainhash.Hash) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("SetBlockProcessed").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:SetBlockProcessed")
	defer span.Finish()

	value := time.Now()
	key := append([]byte("block_processed_"), blockHash[:]...)

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	err := enc.Encode(value)
	if err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to encode data: %w", err)
	}

	if err = s.store.Update(func(tx *badger.Txn) error {
		return tx.Set(key, data.Bytes())
	}); err != nil {
		span.SetTag(string(ext.Error), true)
		span.LogFields(log.Error(err))
		return fmt.Errorf("failed to set data: %w", err)
	}

	return nil
}

func (s *Badger) Ping(ctx context.Context) error {
	start := gocore.CurrentNanos()
	defer func() {
		gocore.NewStat("mtm_store_badger").NewStat("Ping").AddTime(start)
	}()
	span, _ := opentracing.StartSpanFromContext(ctx, "badger:Ping")
	defer span.Finish()

	if s.store == nil {
		return errors.New("badger db store not found")
	}

	return nil
}

func (p *Badger) ClearData(ctx context.Context, retentionDays int32) (int64, error) {
	// Todo: implement function for clearing data
	return 0, errors.New("not implemented")
}
