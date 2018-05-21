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
	ErrInvalidLength   = errors.New("script does not meet the minimum or maximum length requirement")
	ErrInvalidScript   = errors.New("invalid script")
	ErrUnknownCommand  = errors.New("unknown command")
	ErrInvalidPushData = errors.New("invalid pushdata")
)

const (
	FlagByte      = 0x9F
	MinScriptSize = 1 + 1 + 2 + 1 + 32
	MaxScriptSize = 220
	HashSize      = 32
)

type Command byte

func (c *Command) String() string {
	if *c == AddFileCommand {
		return "AddFile"
	} else {
		return "Vote"
	}
}

const (
	AddFileCommand Command = 0x01
	VoteCommand    Command = 0x02
)

type DataType byte

const (
	Cid         DataType = 0x00
	Description DataType = 0x01
	Txid        DataType = 0x02
	Vote        DataType = 0x03
	Comment     DataType = 0x04
	Category    DataType = 0x05
)

type Script interface {
	Command() Command
	ID() []byte
	Parsed() ParsedScript
	Serialize() ([]byte, error)
}

type AddFileScript struct {
	Cid         cid.Cid
	Description string
	Category    string
}

func (as *AddFileScript) Command() Command {
	return AddFileCommand
}

func (as *AddFileScript) ID() []byte {
	return as.Cid.Bytes()
}

func (as *AddFileScript) Parsed() ParsedScript {
	return ParsedScript{
		Description: as.Description,
		Cid:         as.Cid,
		Category:    as.Category,
	}
}

func (as *AddFileScript) Serialize() ([]byte, error) {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData([]byte{FlagByte, byte(AddFileCommand)})
	builder.AddData(append([]byte{byte(Cid)}, as.Cid.Bytes()...))
	if as.Description != "" {
		builder.AddData(append([]byte{byte(Description)}, []byte(as.Description)...))
	}
	if as.Category != "" {
		builder.AddData(append([]byte{byte(Category)}, []byte(as.Category)...))
	}
	return builder.Script()
}

type VoteScript struct {
	Txid    chainhash.Hash
	Comment string
	Upvote  bool
}

func (vs *VoteScript) Command() Command {
	return VoteCommand
}

func (vs *VoteScript) ID() []byte {
	return vs.Txid.CloneBytes()
}

func (vs *VoteScript) Parsed() ParsedScript {
	return ParsedScript{
		Txid:    vs.Txid,
		Comment: vs.Comment,
		Upvote:  vs.Upvote,
	}
}

func (vs *VoteScript) Serialize() ([]byte, error) {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_RETURN)
	builder.AddData([]byte{FlagByte, byte(VoteCommand)})
	txid, err := toBigEndian(&vs.Txid)
	if err != nil {
		return []byte{}, err
	}
	builder.AddData(append([]byte{byte(Txid)}, txid...))
	v := txscript.OP_0
	if vs.Upvote {
		v = txscript.OP_1
	}
	builder.AddData([]byte{byte(Vote), byte(v)})
	builder.AddData(append([]byte{byte(Comment)}, []byte(vs.Comment)...))
	script, err := builder.Script()
	if err != nil {
		return []byte{}, err
	}
	if len(script) > MaxScriptSize {
		return []byte{}, ErrInvalidLength
	}
	return script, nil
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
	case AddFileCommand:
		ps, err := parseDataElements(buf)
		if err != nil {
			return nil, err
		}
		s = &AddFileScript{
			Cid:         ps.Cid,
			Description: ps.Description,
			Category:    ps.Category,
		}
	case VoteCommand:
		ps, err := parseDataElements(buf)
		if err != nil {
			return nil, err
		}
		s = &VoteScript{
			Txid:    ps.Txid,
			Comment: ps.Comment,
			Upvote:  ps.Upvote,
		}
	default:
		return nil, ErrUnknownCommand
	}
	return s, nil
}

type ParsedScript struct {
	Cid         cid.Cid
	Description string
	Txid        chainhash.Hash
	Upvote      bool
	Comment     string
	Category    string
}

func parseDataElements(buf *bytes.Buffer) (ParsedScript, error) {
	var ps ParsedScript
	for buf.Len() > 1 {
		data, err := parsePushData(buf)
		if err != nil {
			return ps, err
		}
		switch DataType(data[0]) {
		case Cid:
			c, err := cid.Cast(data[1:])
			if err != nil {
				return ps, err
			}
			ps.Cid = *c
		case Description:
			ps.Description = string(data[1:])
		case Txid:
			ch, err := fromBigEndian(data[1:])
			if err != nil {
				return ps, err
			}
			ps.Txid = *ch
		case Vote:
			ps.Upvote = byte(data[1]) > 0x00
		case Comment:
			ps.Comment = string(data[1:])
		case Category:
			ps.Category = string(data[1:])
		}
	}
	if buf.Len() != 0 {
		return ps, ErrInvalidScript
	}
	return ps, nil
}

func parsePushData(script *bytes.Buffer) (ret []byte, err error) {
	l, err := script.ReadByte()
	if err != nil {
		return ret, err
	}
	length := int(l)
	if l < 1 {
		return nil, ErrInvalidPushData
	}
	if l == 0x4c {
		l, err := script.ReadByte()
		if err != nil {
			return ret, err
		}
		length = int(l)
	} else if l > 0x4c {
		return nil, ErrInvalidPushData
	}

	if script.Len() < 1 || script.Len() < length {
		return ret, ErrInvalidPushData
	}
	return script.Next(length), nil
}

func evalByte(buf *bytes.Buffer, check byte) (bool, error) {
	b, err := buf.ReadByte()
	if err != nil {
		return false, err
	}
	return b == check, nil
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
