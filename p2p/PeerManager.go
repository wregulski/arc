package p2p

import (
	"fmt"
	"sync"
	"time"

	"github.com/TAAL-GmbH/arc/metamorph/metamorph_api"
	"github.com/TAAL-GmbH/arc/metamorph/store"
	"github.com/TAAL-GmbH/arc/p2p/chaincfg/chainhash"
	"github.com/TAAL-GmbH/arc/p2p/wire"
	"github.com/libsv/go-bt/v2"
	"github.com/ordishs/go-utils"

	"github.com/ordishs/go-utils/batcher"
	"github.com/ordishs/gocore"
)

type PeerManager struct {
	mu         sync.RWMutex
	peers      map[string]*Peer
	invBatcher *batcher.Batcher[[]byte]
}

type PMMessage struct {
	Start  time.Time
	Txid   string
	Status metamorph_api.Status
	Err    error
}

func NewPeerManager(s store.Store, messageCh chan *PMMessage) PeerManagerI {

	pm := &PeerManager{
		peers: make(map[string]*Peer),
	}

	pm.invBatcher = batcher.New(500, 500*time.Millisecond, pm.sendInvBatch, true)

	peerCount, _ := gocore.Config().GetInt("peerCount", 0)
	if peerCount == 0 {
		logger.Fatalf("peerCount must be set")
	}

	for i := 1; i <= peerCount; i++ {
		p2pURL, err, found := gocore.Config().GetURL(fmt.Sprintf("peer_%d_p2p", i))
		if !found {
			logger.Fatalf("peer_%d_p2p must be set", i)
		}
		if err != nil {
			logger.Fatalf("Error reading peer_%d_p2p: %v", i, err)
		}

		peer, err := NewPeer(p2pURL.Host, s, messageCh)
		if err != nil {
			logger.Fatalf("Error creating peer: %v", err)
		}

		pm.addPeer(peer)
	}

	return pm
}

func (pm *PeerManager) addPeer(peer *Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.peers[peer.address] = peer
}

func (pm *PeerManager) AnnounceNewTransaction(txID []byte) {
	pm.invBatcher.Put(&txID)
}

func (pm *PeerManager) sendInvBatch(batch []*[]byte) {
	invMsg := wire.NewMsgInvSizeHint(uint(len(batch)))

	for _, txid := range batch {
		hash, err := chainhash.NewHash(*txid)
		if err != nil {
			logger.Infof("ERROR announcing new tx [%x]: %v", txid, err)
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeTx, hash)
		_ = invMsg.AddInvVect(iv)
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, peer := range pm.peers {
		utils.SafeSend[wire.Message](peer.writeChan, invMsg)
	}

	// if len(batch) <= 10 {
	logger.Infof("Sent INV (%d items) to %d peers", len(batch), len(pm.peers))
	for _, txid := range batch {
		logger.Infof("        %x", bt.ReverseBytes(*txid))
	}
	// } else {
	// 	logger.Infof("Sent INV (%d items) to %d peers", len(batch), len(pm.peers))
	// }
}
