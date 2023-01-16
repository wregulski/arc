package p2p

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TAAL-GmbH/arc/p2p/bsvutil"
	"github.com/TAAL-GmbH/arc/p2p/wire"

	"github.com/ordishs/go-utils"
	"github.com/ordishs/gocore"
)

var (
	pingInterval = 2 * time.Minute
)

type Block struct {
	Hash         []byte `json:"hash,omitempty"`          // Little endian
	PreviousHash []byte `json:"previous_hash,omitempty"` // Little endian
	MerkleRoot   []byte `json:"merkle_root,omitempty"`   // Little endian
	Height       uint64 `json:"height,omitempty"`
}

type Peer struct {
	address        string
	network        wire.BitcoinNet
	mu             sync.RWMutex
	readConn       net.Conn
	writeConn      net.Conn
	peerHandler    PeerHandlerI
	writeChan      chan wire.Message
	quit           chan struct{}
	logger         utils.Logger
	sentVerAck     atomic.Bool
	receivedVerAck atomic.Bool
}

// NewPeer returns a new bitcoin peer for the provided address and configuration.
func NewPeer(logger utils.Logger, address string, peerHandler PeerHandlerI, network wire.BitcoinNet) (*Peer, error) {
	writeChan := make(chan wire.Message, 100)

	p := &Peer{
		network:     network,
		address:     address,
		writeChan:   writeChan,
		peerHandler: peerHandler,
		logger:      logger,
	}

	go p.pingHandler()
	go p.writeChannelHandler()

	// reconnect if disconnected
	go func() {
		for {
			if !p.Connected() {
				err := p.connect()
				if err != nil {
					logger.Warnf("Failed to connect to peer %s: %v", address, err)
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return p, nil
}

func (p *Peer) disconnect() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.readConn != nil {
		_ = p.readConn.Close()
	}

	p.readConn = nil
	p.writeConn = nil
	p.sentVerAck.Store(false)
	p.receivedVerAck.Store(false)
}

func (p *Peer) connect() error {
	p.mu.Lock()
	p.readConn = nil
	p.sentVerAck.Store(false)
	p.receivedVerAck.Store(false)
	p.mu.Unlock()

	p.logger.Infof("[%s] Connecting to peer on %s", p.address, p.network)
	conn, err := net.Dial("tcp", p.address)
	if err != nil {
		return fmt.Errorf("could not dial node [%s]: %v", p.address, err)
	}

	// open the read connection, so we can receive messages
	p.mu.Lock()
	p.readConn = conn
	p.mu.Unlock()

	go p.readHandler()

	// write version message to our peer directly and not through the write channel,
	// write channel is not ready to send message until the VERACK handshake is done
	msg := p.versionMessage(p.address)

	if err = wire.WriteMessage(conn, msg, wire.ProtocolVersion, p.network); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}
	p.logger.Debugf("[%s] Sent %s", p.address, strings.ToUpper(msg.Command()))

	for {
		if p.receivedVerAck.Load() && p.sentVerAck.Load() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// set the connection which allows us to send messages
	p.mu.Lock()
	p.writeConn = conn
	p.mu.Unlock()

	return nil
}

func (p *Peer) Connected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.readConn != nil && p.writeConn != nil
}

func (p *Peer) WriteMsg(msg wire.Message) error {
	p.mu.RLock()
	writeConn := p.writeConn
	p.mu.RUnlock()

	if writeConn == nil {
		return errors.New("peer is not connected")
	}

	p.writeChan <- msg
	return nil
}

func (p *Peer) String() string {
	return p.address
}

func (p *Peer) readHandler() {
	p.mu.RLock()
	readConn := p.readConn
	p.mu.RUnlock()

	if readConn != nil {
		for {
			msg, b, err := wire.ReadMessage(readConn, wire.ProtocolVersion, p.network)
			if err != nil {
				if errors.Is(err, io.EOF) {
					p.logger.Errorf(fmt.Sprintf("READ EOF whilst reading from %s [%d bytes]\n%s", p.address, len(b), string(b)))
					p.disconnect()
					break
				}
				p.logger.Errorf("[%s] Failed to read message: %v", p.address, err)
				continue
			}

			switch msg.Command() {
			case wire.CmdVersion:
				p.logger.Debugf("[%s] Recv %s", p.address, strings.ToUpper(msg.Command()))
				verackMsg := wire.NewMsgVerAck()
				if err = wire.WriteMessage(readConn, verackMsg, wire.ProtocolVersion, p.network); err != nil {
					p.logger.Errorf("[%s] failed to write message: %v", p.address, err)
				}
				p.logger.Debugf("[%s] Sent %s", p.address, strings.ToUpper(verackMsg.Command()))
				p.sentVerAck.Store(true)

			case wire.CmdPing:
				pingMsg := msg.(*wire.MsgPing)
				p.writeChan <- wire.NewMsgPong(pingMsg.Nonce)

			case wire.CmdInv:
				invMsg := msg.(*wire.MsgInv)
				p.logger.Infof("[%s] Recv INV (%d items)", p.address, len(invMsg.InvList))
				if p.logger.LogLevel() == int(gocore.DEBUG) {
					for _, inv := range invMsg.InvList {
						p.logger.Debugf("        [%s] %s", p.address, inv.Hash.String())
					}
				}

				go func(invList []*wire.InvVect) {
					for _, invVect := range invList {
						switch invVect.Type {
						case wire.InvTypeTx:
							if err = p.peerHandler.HandleTransactionAnnouncement(invVect, p); err != nil {
								p.logger.Errorf("[%s] Unable to process tx %s: %v", p.address, invVect.Hash.String(), err)
							}
						case wire.InvTypeBlock:
							if err = p.peerHandler.HandleBlockAnnouncement(invVect, p); err != nil {
								p.logger.Errorf("[%s] Unable to process block %s: %v", p.address, invVect.Hash.String(), err)
							}
						}
					}
				}(invMsg.InvList)

			case wire.CmdGetData:
				dataMsg := msg.(*wire.MsgGetData)
				p.logger.Infof("[%s] Recv GETDATA (%d items)", p.address, len(dataMsg.InvList))
				if p.logger.LogLevel() == int(gocore.DEBUG) {
					for _, inv := range dataMsg.InvList {
						p.logger.Debugf("        [%s] %s", p.address, inv.Hash.String())
					}
				}
				p.handleGetDataMsg(dataMsg)

			case wire.CmdBlock:
				blockMsg := msg.(*wire.MsgBlock)
				p.logger.Infof("[%s] Recv %s: %s", p.address, strings.ToUpper(msg.Command()), blockMsg.BlockHash().String())

				err = p.peerHandler.HandleBlock(blockMsg, p)
				if err != nil {
					p.logger.Errorf("[%s] Unable to process block %s: %v", p.address, blockMsg.BlockHash().String(), err)
				}

				// read the remainder of the block, if not consumed by the handler
				// TODO is this necessary or can we just ignore whether the reader has been consumed?
				_, _ = io.ReadAll(blockMsg.TransactionReader)

			case wire.CmdReject:
				rejMsg := msg.(*wire.MsgReject)
				if err = p.peerHandler.HandleTransactionRejection(rejMsg, p); err != nil {
					p.logger.Errorf("[%s] Unable to process block %s: %v", p.address, rejMsg.Hash.String(), err)
				}

			case wire.CmdVerAck:
				p.logger.Debugf("[%s] Recv %s", p.address, strings.ToUpper(msg.Command()))
				p.receivedVerAck.Store(true)

			default:
				p.logger.Debugf("[%s] Ignored %s", p.address, strings.ToUpper(msg.Command()))
			}
		}
	}
}

func (p *Peer) handleGetDataMsg(dataMsg *wire.MsgGetData) {
	for _, invVect := range dataMsg.InvList {
		switch invVect.Type {
		case wire.InvTypeTx:
			p.logger.Debugf("[%s] Request for TX: %s\n", p.address, invVect.Hash.String())

			txBytes, err := p.peerHandler.GetTransactionBytes(invVect)
			if err != nil {
				p.logger.Errorf("[%s] Unable to fetch tx %s from store: %v", p.address, invVect.Hash.String(), err)
				continue
			}

			if txBytes == nil {
				p.logger.Warnf("[%s] Unable to fetch tx %s from store: %v", p.address, invVect.Hash.String(), err)
				continue
			}

			tx, err := bsvutil.NewTxFromBytes(txBytes)
			if err != nil {
				log.Print(err) // Log and handle the error
				continue
			}

			p.writeChan <- tx.MsgTx()

		case wire.InvTypeBlock:
			p.logger.Infof("[%s] Request for Block: %s\n", p.address, invVect.Hash.String())

		default:
			p.logger.Warnf("[%s] Unknown type: %d\n", p.address, invVect.Type)
		}
	}
}

func (p *Peer) writeChannelHandler() {
	for msg := range p.writeChan {
		// wait for the write connection to be ready
		for {
			p.mu.RLock()
			writeConn := p.writeConn
			p.mu.RUnlock()

			if writeConn != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if err := wire.WriteMessage(p.writeConn, msg, wire.ProtocolVersion, p.network); err != nil {
			if errors.Is(err, io.EOF) {
				panic("WRITE EOF")
			}
			p.logger.Errorf("[%s] Failed to write message: %v", p.address, err)
		}

		if msg.Command() == wire.CmdTx {
			hash := msg.(*wire.MsgTx).TxHash()
			if err := p.peerHandler.HandleTransactionSent(msg.(*wire.MsgTx), p); err != nil {
				p.logger.Errorf("[%s] Unable to process tx %s: %v", p.address, hash.String(), err)
			}
		}

		switch m := msg.(type) {
		case *wire.MsgTx:
			p.logger.Debugf("[%s] Sent %s: %s", p.address, strings.ToUpper(msg.Command()), m.TxHash().String())
		case *wire.MsgBlock:
			p.logger.Debugf("[%s] Sent %s: %s", p.address, strings.ToUpper(msg.Command()), m.BlockHash().String())
		case *wire.MsgGetData:
			p.logger.Debugf("[%s] Sent %s: %s", p.address, strings.ToUpper(msg.Command()), m.InvList[0].Hash.String())
		case *wire.MsgInv:
		default:
			p.logger.Debugf("[%s] Sent %s", p.address, strings.ToUpper(msg.Command()))
		}
	}
}

func (p *Peer) versionMessage(address string) *wire.MsgVersion {
	lastBlock := int32(0)

	tcpAddrMe := &net.TCPAddr{IP: nil, Port: 0}
	me := wire.NewNetAddress(tcpAddrMe, wire.SFNodeNetwork)

	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		panic(fmt.Sprintf("Could not parse address %s", address))
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(fmt.Sprintf("Could not parse port %s", parts[1]))
	}

	tcpAddrYou := &net.TCPAddr{IP: net.ParseIP(parts[0]), Port: port}
	you := wire.NewNetAddress(tcpAddrYou, wire.SFNodeNetwork)

	nonce, err := wire.RandomUint64()
	if err != nil {
		p.logger.Errorf("[%s] RandomUint64: error generating nonce: %v", p.address, err)
	}

	msg := wire.NewMsgVersion(me, you, nonce, lastBlock)

	return msg
}

// pingHandler periodically pings the peer.  It must be run as a goroutine.
func (p *Peer) pingHandler() {
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

out:
	for {
		select {
		case <-pingTicker.C:
			nonce, err := wire.RandomUint64()
			if err != nil {
				p.logger.Errorf("[%s] Not sending ping to %s: %v", p.address, p, err)
				continue
			}
			p.writeChan <- wire.NewMsgPing(nonce)

		case <-p.quit:
			break out
		}
	}
}
