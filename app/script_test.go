package app

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"testing"
)

type TestScript struct {
	script        string
	data          string
	expectedError error
}

var testScripts = []TestScript{
	{
		"6a029F012212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b2390b68656c6c6f20776f726c64",
		"hello world",
		nil,
	},
	{
		"6a029F02200934aaa9e475375cea77c01853d6c411e6c4446c81da76797f696fd70e143cc3510D676f6f6462796520776f726c64",
		"goodbye world",
		nil,
	},
	{
		"6b029F022212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		"",
		ErrInvalidScript,
	},
	{
		"6a029F0222122a",
		"",
		ErrInvalidLength,
	},
	{
		"6a029F0222122a6a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c646a029F032212200709a33d6f07812bc1d7cbddb6a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64bc2f95f4444f5d0cf5deb05a441c4b21fc6b23906a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c646a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c6410b68656c6c6f20776f726c64",
		"",
		ErrInvalidLength,
	},
	{
		"6a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		"",
		ErrUnknownCommand,
	},
	{
		"6a0294032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		"",
		ErrInvalidScript,
	},
	{
		"6a049F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		"",
		ErrInvalidScript,
	},
}

func TestParseScript(t *testing.T) {
	for i, test := range testScripts {
		script, err := hex.DecodeString(test.script)
		if err != nil {
			t.Error(err)
		}
		returnedScript, err := ParseScript(script)
		if err != test.expectedError {
			t.Errorf("Test script %d failed: %s", i, err.Error())
		}
		if test.data != "" && test.data != returnedScript.Data() {
			fmt.Println(returnedScript.Data())
			t.Errorf("Test script %d return incorrect data", i)
		}
	}
}

func TestVoteScript_Serialize(t *testing.T) {
	ch, err := chainhash.NewHashFromStr("0934aaa9e475375cea77c01853d6c411e6c4446c81da76797f696fd70e143cc3")
	if err != nil {
		t.Error(err)
	}
	script := VoteScript{
		Txid:    *ch,
		Upvote:  true,
		Comment: "goodbye world",
	}
	check, err := hex.DecodeString("6a029F02200934aaa9e475375cea77c01853d6c411e6c4446c81da76797f696fd70e143cc3510D676f6f6462796520776f726c64")
	if err != nil {
		t.Error(err)
	}
	ser, err := script.Serialize()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(ser, check) {
		t.Error("failed to serialize properly")
	}
}

func TestAddFileScript_Serialize(t *testing.T) {
	id, err := cid.Decode("QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ")
	if err != nil {
		t.Error(err)
	}
	script := AddFileScript{
		Cid:         *id,
		Description: "hello world",
	}
	check, err := hex.DecodeString("6a029F012212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b2390b68656c6c6f20776f726c64")
	if err != nil {
		t.Error(err)
	}
	ser, err := script.Serialize()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(ser, check) {
		t.Error("failed to serialize properly")
	}
}
