package app

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
)

var (
	ErrInvalidLength   = errors.New("script does not meet the minimum length")
	ErrInvalidScript   = errors.New("invalid script")
	ErrUnknownCommand  = errors.New("unknown command")
	ErrInvalidPushData = errors.New("invalid pushdata")
)

const (
	FlagByte      = 0x9F
	MinScriptSize = 1 + 1 + 2 + 1 + 34
	MaxScriptSize = 220
	HashSize      = 32
)

type Command byte

func (c *Command) String() string {
	if *c == AddFile {
		return "AddFile"
	} else {
		return "Vote"
	}
}

const (
	AddFile Command = 0x01
	Vote    Command = 0x02
)

type Script interface {
	Command() Command
	ID() []byte
	Data() string
	Serialize() ([]byte, error)
}

type AddFileScript struct {
	Cid         cid.Cid
	Description string
}

func (as *AddFileScript) Command() Command {
	return AddFile
}

func (as *AddFileScript) ID() []byte {
	return as.Cid.Bytes()
}

func (as *AddFileScript) Data() string {
	return as.Description
}

func (as *AddFileScript) Serialize() ([]byte, error) {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData([]byte{FlagByte, byte(AddFile)})
	builder.AddData(as.Cid.Bytes())
	builder.AddData([]byte(as.Description))
	return builder.Script()
}

type VoteScript struct {
	Txid    chainhash.Hash
	Comment string
	Upvote  bool
}

func (vs *VoteScript) Command() Command {
	return Vote
}

func (vs *VoteScript) ID() []byte {
	return vs.Txid.CloneBytes()
}

func (vs *VoteScript) Data() string {
	return vs.Comment
}

func (vs *VoteScript) Serialize() ([]byte, error) {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData([]byte{FlagByte, byte(Vote)})
	txid, err := toBigEndian(&vs.Txid)
	if err != nil {
		return []byte{}, err
	}
	builder.AddData(txid)
	v := txscript.OP_0
	if vs.Upvote {
		v = txscript.OP_1
	}
	builder.AddOp(byte(v))
	builder.AddData([]byte(vs.Comment))
	return builder.Script()
}

func ParseScript(script []byte) (Script, error) {
	buf := bytes.NewBuffer(script)
	if buf.Len() < MinScriptSize || buf.Len() > MaxScriptSize {
		return nil, ErrInvalidLength
	}
	if ok, err := evalByte(buf, txscript.OP_RETURN); !ok || err != nil {
		return nil, ErrInvalidScript
	}

	if ok, err := evalByte(buf, txscript.OP_DATA_2); !ok || err != nil {
		return nil, ErrInvalidScript
	}

	if ok, err := evalByte(buf, FlagByte); !ok || err != nil {
		return nil, ErrInvalidScript
	}

	b, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	var s Script
	switch Command(b) {
	case AddFile:
		id, err := extractCid(buf)
		if err != nil {
			return nil, err
		}
		description, err := parsePushData(buf)
		if err != nil {
			return nil, err
		}
		s = &AddFileScript{
			Cid:         id,
			Description: string(description),
		}
	case Vote:
		txidBytes, err := parsePushData(buf)
		if err != nil {
			return nil, err
		}
		txid, err := fromBigEndian(txidBytes)
		if err != nil {
			return nil, err
		}
		v, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		comment, err := parsePushData(buf)
		if err != nil {
			return nil, err
		}
		s = &VoteScript{
			Txid:    *txid,
			Comment: string(comment),
			Upvote:  int(v) > 0,
		}
	default:
		return nil, ErrUnknownCommand
	}
	return s, nil
}

func evalByte(buf *bytes.Buffer, check byte) (bool, error) {
	b, err := buf.ReadByte()
	if err != nil {
		return false, err
	}
	return b == check, nil
}

func extractCid(script *bytes.Buffer) (cid.Cid, error) {
	cidBytes, err := parsePushData(script)
	if err != nil {
		return cid.Cid{}, err
	}
	c, err := cid.Cast(cidBytes)
	if err != nil {
		return cid.Cid{}, err
	}
	return *c, nil
}

func parsePushData(script *bytes.Buffer) (ret []byte, err error) {
	l, err := script.ReadByte()
	if err != nil {
		return ret, err
	}
	if l < 1 || l > 74 {
		return nil, ErrInvalidPushData
	}
	if script.Len() < 1 || script.Len() < int(l) {
		return ret, ErrInvalidPushData
	}
	return script.Next(int(l)), nil
}

func toBigEndian(txid *chainhash.Hash) ([]byte, error) {
	return hex.DecodeString(txid.String())
}

func fromBigEndian(hash []byte) (*chainhash.Hash, error) {
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return chainhash.NewHash(hash)
}
