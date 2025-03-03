package metamorph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bitcoin-sv/arc/api"
	"github.com/bitcoin-sv/arc/metamorph/store"
	"github.com/ordishs/go-utils"
)

const (
	CallbackTries           = 5
	CallbackIntervalSeconds = 5
)

func SendCallback(logger *slog.Logger, tx *store.StoreData) {
	sleepDuration := CallbackIntervalSeconds
	for i := 0; i < CallbackTries; i++ {
		statusString := tx.Status.String()
		blockHash := ""
		if tx.BlockHash != nil {
			blockHash = utils.ReverseAndHexEncodeSlice(tx.BlockHash.CloneBytes())
		}

		logger.Info("Sending callback for transaction", slog.String("hash", tx.Hash.String()), slog.String("url", tx.CallbackUrl), slog.String("token", tx.CallbackToken), slog.String("status", statusString), slog.Uint64("block height", tx.BlockHeight), slog.String("block hash", blockHash))

		status := &api.TransactionStatus{
			BlockHash:   &blockHash,
			BlockHeight: &tx.BlockHeight,
			TxStatus:    &statusString,
			Txid:        tx.Hash.String(),
			Timestamp:   time.Now(),
		}
		statusBytes, err := json.Marshal(status)
		if err != nil {
			logger.Error("Couldn't marshal status", slog.String("err", err.Error()))
			return
		}

		var request *http.Request
		request, err = http.NewRequest("POST", tx.CallbackUrl, bytes.NewBuffer(statusBytes))
		if err != nil {
			logger.Error("Couldn't marshal status", slog.String("url", tx.CallbackUrl), slog.String("token", tx.CallbackToken), slog.String("hash", tx.Hash.String()), slog.String("err", errors.Join(err, fmt.Errorf("failed to post callback for transaction id %s", tx.Hash)).Error()))
			return
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		if tx.CallbackToken != "" {
			request.Header.Set("Authorization", "Bearer "+tx.CallbackToken)
		}

		// default http client
		httpClient := http.Client{
			Timeout: 5 * time.Second,
		}

		var response *http.Response
		response, err = httpClient.Do(request)
		if err != nil {
			logger.Error("Couldn't send transaction info through callback url", slog.String("url", tx.CallbackUrl), slog.String("token", tx.CallbackToken), slog.String("hash", tx.Hash.String()), slog.String("err", err.Error()))
			continue
		}
		defer response.Body.Close()

		// if callback was sent successfully we stop here
		if response.StatusCode == http.StatusOK {
			// err = s.RemoveCallbacker(context.Background(), tx.Hash)
			// if err != nil {
			// 	logger.Error("Couldn't update/remove callback url", slog.String("err", err.Error()))
			// 	return
			// }
			return
		}

		logger.Error("Callback response status code not ok", slog.String("url", tx.CallbackUrl), slog.String("token", tx.CallbackToken), slog.String("hash", tx.Hash.String()), slog.Int("status", response.StatusCode))

		// sleep before trying again
		time.Sleep(time.Duration(sleepDuration) * time.Second)
		// increase intervals on each failure
		sleepDuration *= 2
	}

	logger.Error("Couldn't send transaction info through callback url after tries", slog.String("url", tx.CallbackUrl), slog.String("token", tx.CallbackToken), slog.String("hash", tx.Hash.String()), slog.Int("retries", CallbackTries))
}
