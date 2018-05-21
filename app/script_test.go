package app

import (
	"bytes"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"testing"
)

type TestScript struct {
	script        string
	data          ParsedScript
	cidHex        string
	txidHex       string
	expectedError error
}

var testScripts = []TestScript{
	{
		"6a029F01230012200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b2390c0168656c6c6f20776f726c64",
		ParsedScript{
			Description: "hello world",
		},
		"12200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239",
		"",
		nil,
	},
	{
		"6a029F01230012200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b2390c0168656c6c6f20776f726c6406054d75736963",
		ParsedScript{
			Description: "hello world",
			Category:    "Music",
		},
		"12200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239",
		"",
		nil,
	},
	{
		"6a029F0221020934aaa9e475375cea77c01853d6c411e6c4446c81da76797f696fd70e143cc30203510E04676f6f6462796520776f726c64",
		ParsedScript{
			Comment: "goodbye world",
			Upvote:  true,
		},
		"",
		"0934aaa9e475375cea77c01853d6c411e6c4446c81da76797f696fd70e143cc3",
		nil,
	},
	{
		script:        "6b029F022212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		expectedError: ErrInvalidScript,
	},
	{
		script:        "6a029F0222122a",
		expectedError: ErrInvalidLength,
	},
	{
		script:        "6a029F0222122a6a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c646a029F032212200709a33d6f07812bc1d7cbddb6a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64bc2f95f4444f5d0cf5deb05a441c4b21fc6b23906a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c646a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c6410b68656c6c6f20776f726c64",
		expectedError: ErrInvalidLength,
	},
	{
		script:        "6a029F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		expectedError: ErrUnknownCommand,
	},
	{
		script:        "6a0294032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		expectedError: ErrInvalidScript,
	},
	{
		script:        "6a049F032212200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b239010b68656c6c6f20776f726c64",
		expectedError: ErrInvalidScript,
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
			continue
		}
		if err == nil {
			if test.cidHex != "" {
				b, err := hex.DecodeString(test.cidHex)
				if err != nil {
					t.Error(err)
				}
				c, err := cid.Cast(b)
				if err != nil {
					t.Error(err)
				}
				test.data.Cid = *c
			}
			if test.txidHex != "" {
				ch, err := chainhash.NewHashFromStr(test.txidHex)
				if err != nil {
					t.Error(err)
				}
				test.data.Txid = *ch
			}

			r1 := returnedScript.Parsed()
			r2 := test.data
			if r1.Cid.String() != r2.Cid.String() {
				t.Errorf("Test script %d parsed incorrectly", i)
			}
			if returnedScript.Parsed().Txid.String() != test.data.Txid.String() {
				t.Errorf("Test script %d parsed incorrectly", i)
			}
			if returnedScript.Parsed().Upvote != test.data.Upvote {
				t.Errorf("Test script %d parsed incorrectly", i)
			}
			if returnedScript.Parsed().Description != test.data.Description {
				t.Errorf("Test script %d parsed incorrectly", i)
			}
			if returnedScript.Parsed().Comment != test.data.Comment {
				t.Errorf("Test script %d parsed incorrectly", i)
			}
			if returnedScript.Parsed().Category != test.data.Category {
				t.Errorf("Test script %d parsed incorrectly", i)
			}
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
	check, err := hex.DecodeString("6a029F0221020934aaa9e475375cea77c01853d6c411e6c4446c81da76797f696fd70e143cc30203510E04676f6f6462796520776f726c64")
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
	check, err := hex.DecodeString("6a029F01230012200709a33d6f07812bc1d7cbddbbc2f95f4444f5d0cf5deb05a441c4b21fc6b2390c0168656c6c6f20776f726c64")
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

func Test(t *testing.T) {
	h := "6a029f0123001220627a32cf4b279ccf1c6d636485ba7483870eba69fa554cbfafc906f4c463b2c24c5a01536e6f77666c616b6520746f204176616c616e6368653a2041204e6f76656c204d657461737461626c6520436f6e73656e7375732050726f746f636f6c2046616d696c7920666f722043727970746f63757272656e63696573100541636164656d696320506170657273"

	hd, _ := hex.DecodeString(h)

	_, err := ParseScript(hd)
	if err != nil {
		t.Error(err)
	}

}