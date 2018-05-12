package app

import (
	"errors"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	btc "github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/txsort"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/cpacia/BitcoinCash-Wallet"
	"github.com/cpacia/bchutil"
)

func MakeTransaction(w *bitcoincash.SPVWallet, utxos []wallet.Utxo, ipfsScript Script) (*chainhash.Hash, error) {
	var val int64
	var inputs []*wire.TxIn
	additionalPrevScripts := make(map[wire.OutPoint][]byte)
	for _, u := range utxos {
		val += u.Value
		in := wire.NewTxIn(&u.Op, []byte{}, [][]byte{})
		inputs = append(inputs, in)
		additionalPrevScripts[u.Op] = u.ScriptPubkey
	}
	serializedIPFSScript, err := ipfsScript.Serialize()
	if err != nil {
		return nil, err
	}
	ipfsOutput := wire.NewTxOut(0, serializedIPFSScript)

	estimatedSize := bitcoincash.EstimateSerializeSize(len(utxos), []*wire.TxOut{ipfsOutput}, true, bitcoincash.P2PKH)
	estimatedSize += len(serializedIPFSScript)

	// Calculate the fee
	feePerByte := int(w.GetFeePerByte(wallet.ECONOMIC))
	fee := estimatedSize * feePerByte

	outVal := val - int64(fee)
	if outVal < 0 {
		outVal = 0
	}

	tx := &wire.MsgTx{
		Version:  wire.TxVersion,
		TxIn:     inputs,
		TxOut:    []*wire.TxOut{ipfsOutput},
		LockTime: 0,
	}

	// Check for dust. If we are over threshold send as change.
	internalAddr := w.CurrentAddress(wallet.INTERNAL)
	changeScript, _ := bchutil.PayToAddrScript(internalAddr)
	if !txrules.IsDustAmount(btc.Amount(outVal), len(changeScript), txrules.DefaultRelayFeePerKb) {
		changeOut := wire.NewTxOut(outVal, changeScript)
		tx.TxOut = append(tx.TxOut, changeOut)
	}

	// BIP 69 sorting
	txsort.InPlaceSort(tx)

	getKey := txscript.KeyClosure(func(addr btc.Address) (*btcec.PrivateKey, bool, error) {
		key, err := w.GetKey(addr)
		if err != nil {
			return nil, false, err
		}
		wif, err := btc.NewWIF(key, w.Params(), true)
		if err != nil {
			return nil, false, err
		}
		return wif.PrivKey, wif.CompressPubKey, nil
	})
	getScript := txscript.ScriptClosure(func(addr btc.Address) ([]byte, error) {
		return []byte{}, nil
	})
	for i, txIn := range tx.TxIn {
		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
		script, err := bchutil.SignTxOutput(w.Params(),
			tx, i, prevOutScript, txscript.SigHashAll, getKey,
			getScript, txIn.SignatureScript, utxos[i].Value)
		if err != nil {
			log.Error(err)
			return nil, errors.New("Failed to sign transaction")
		}
		txIn.SignatureScript = script
	}

	// broadcast
	w.Broadcast(tx)
	txid := tx.TxHash()
	return &txid, nil
}
