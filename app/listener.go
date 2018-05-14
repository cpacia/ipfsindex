package app

import (
	"encoding/xml"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/cpacia/BitcoinCash-Wallet"
	"github.com/cpacia/ipfsindex/db"
	"github.com/jinzhu/gorm"
	"sync"
	"time"
	"strings"
	"github.com/microcosm-cc/bluemonday"
)

type UserEntry struct {
	ID          string
	Script      Script
	Address     btcutil.Address
	Timestamp   time.Time
	AmountToPay uint64
	AmountPaid  uint64
}

type TransactionListener struct {
	UserEntries map[string]UserEntry
	wallet      *bitcoincash.SPVWallet
	db          *db.Database
	addrChan    chan string
	lock        sync.RWMutex
}

func NewTransactionListener(wallet *bitcoincash.SPVWallet, db *db.Database, addrChan chan string) *TransactionListener {
	tl := &TransactionListener{make(map[string]UserEntry), wallet, db, addrChan, sync.RWMutex{}}
	ticker := time.NewTicker(time.Minute)
	go func() {
		select {
		case <-ticker.C:
			tl.cleanup()
		}
	}()
	return tl
}

func (l *TransactionListener) ListenBitcoinCash(tx wallet.TransactionCallback) {
	entries := make(map[UserEntry][]wallet.Utxo)
	chainHash, err := chainhash.NewHash(tx.Txid)
	if err != nil {
		log.Error(err)
		return
	}
	for _, out := range tx.Outputs {
		addr, err := l.wallet.ScriptToAddress(out.ScriptPubKey)
		if err != nil {
			parsedScript, err := ParseScript(out.ScriptPubKey)
			if err != nil {
				continue
			}
			if parsedScript.Command() == AddFile {
				ts := time.Now()
				if tx.Height > 0 {
					ts = tx.BlockTime
				}
				fd := &db.FileDescriptor{}
				if l.db.Where("txid = ?", chainHash.String()).First(fd).RecordNotFound() {
					l.db.Save(&db.FileDescriptor{
						Txid:        chainHash.String(),
						Category:    getCategory(parsedScript.(*AddFileScript).Description),
						Description: removeTags(parsedScript.(*AddFileScript).Description),
						Timestamp:   ts,
						Height:      uint32(tx.Height),
						Cid:         parsedScript.(*AddFileScript).Cid.String(),
					})
					log.Debugf("Received new file descriptor, tx: %s", chainHash.String())
				} else {
					l.db.Model(fd).Updates(&db.FileDescriptor{Height: uint32(tx.Height), Timestamp: ts})
					log.Debugf("Updated file descriptor with confirmation, tx: %s", chainHash.String())
				}
			} else if parsedScript.Command() == Vote {
				ts := time.Now()
				if tx.Height > 0 {
					ts = tx.BlockTime
				}
				v := &db.Vote{}
				if l.db.Where("txid = ?", chainHash.String()).First(v).RecordNotFound() {
					l.db.Save(&db.Vote{
						FDTxid:    parsedScript.(*VoteScript).Txid.String(),
						Txid:      chainHash.String(),
						Comment:   parsedScript.(*VoteScript).Comment,
						Timestamp: ts,
						Height:    uint32(tx.Height),
						Upvote:    parsedScript.(*VoteScript).Upvote,
					})
					if tx.Height > 0 {
						l.updateVoteColumns(parsedScript.(*VoteScript).Upvote, parsedScript.(*VoteScript).Txid.String())
					}
					log.Debugf("Received new vote, tx: %s", chainHash.String())
				} else {
					l.db.Model(v).Updates(&db.Vote{Height: uint32(tx.Height), Timestamp: ts})
					l.updateVoteColumns(v.Upvote, v.FDTxid)
					log.Debugf("Updated vote with confirmation, tx: %s", chainHash.String())
				}
			}
			continue
		}
		l.lock.RLock()
		entry, ok := l.UserEntries[addr.String()]
		l.lock.RUnlock()
		if !ok {
			continue
		}
		op := wire.NewOutPoint(chainHash, out.Index)
		u := wallet.Utxo{
			Value:        out.Value,
			ScriptPubkey: out.ScriptPubKey,
			WatchOnly:    false,
			AtHeight:     tx.Height,
			Op:           *op,
		}
		utxos := entries[entry]
		utxos = append(utxos, u)
		entry.AmountPaid += uint64(out.Value)
		l.lock.Lock()
		l.UserEntries[addr.String()] = entry
		l.lock.Unlock()
		entries[entry] = utxos
		log.Debugf("Received transaction %s for req:%s", chainHash.String(), entry.ID)
	}

	for e, utxos := range entries {
		if e.AmountPaid < e.AmountToPay {
			continue
		}
		go func(e2 UserEntry, utxoList []wallet.Utxo) {
			defer func() {
				l.lock.Lock()
				delete(l.UserEntries, e.Address.String())
				l.lock.Unlock()
			}()
			l.addrChan <- e2.Address.String()
			hash, err := MakeTransaction(l.wallet, utxos, e2.Script)
			if err != nil {
				log.Errorf("Error making transaction: req:%s: %s", e2.ID, err.Error())
				return
			}
			log.Debugf("Successfuly broadcast transaction %s for req:%s", hash.String(), e2.ID)
		}(e, utxos)
	}
}

func (l *TransactionListener) updateVoteColumns(upvote bool, txid string) {
	column := "downvotes"
	sign := "-"
	if upvote {
		column = "upvotes"
		sign = "+"
	}
	l.db.Model(&db.FileDescriptor{}).Where(`txid="`+txid+`"`).UpdateColumn("upvotes", gorm.Expr(column+sign+"1")).UpdateColumn("net", gorm.Expr("net"+sign+"1"))
}

func (l *TransactionListener) NewEntry(addr btcutil.Address, entry UserEntry) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.UserEntries[addr.String()] = entry
}

func (l *TransactionListener) cleanup() {
	l.lock.Lock()
	defer l.lock.Unlock()
	for k, v := range l.UserEntries {
		if v.Timestamp.Add(time.Minute * 10).Before(time.Now()) {
			delete(l.UserEntries, k)
		}
	}
}

type Query struct {
	XMLName xml.Name `xml:"meta"`
	Content    string   `xml:"content,attr"`
	Name   string   `xml:"name,attr"`
}

func getCategory(description string) string {
	var q Query
	err := xml.Unmarshal([]byte(description), &q)
	if err != nil {
		return ""
	}
	if strings.ToLower(q.Name) == "category" {
		return q.Content
	}
	return ""
}

func removeTags(description string) string {
	return bluemonday.UGCPolicy().Sanitize(description)
}
