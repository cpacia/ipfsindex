package app

import (
	"crypto/rand"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/cpacia/BitcoinCash-Wallet"
	"github.com/cpacia/BitcoinCash-Wallet/db"
	"github.com/mitchellh/go-homedir"
	"github.com/natefinch/lumberjack"
	"github.com/op/go-logging"
	"github.com/tyler-smith/go-bip39"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

var log = logging.MustGetLogger("app")

var fileLogFormat = logging.MustStringFormatter(
	`%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`,
)

func NewWallet(params *chaincfg.Params, repoPath string, trustedPeer net.Addr) (*bitcoincash.SPVWallet, error) {
	config := bitcoincash.NewDefaultConfig()
	config.Params = params
	if trustedPeer != nil {
		config.TrustedPeer = trustedPeer
	}

	config.RepoPath = repoPath
	if params.Name == chaincfg.TestNet3Params.Name {
		config.RepoPath = path.Join(config.RepoPath, "testnet")
	} else if params.Name == chaincfg.RegressionNetParams.Name {
		config.RepoPath = path.Join(config.RepoPath, "regtest")
	}

	config.AdditionalFilters = [][]byte{
		{FlagByte, byte(AddFileCommand)},
		{FlagByte, byte(VoteCommand)},
	}

	os.Mkdir(config.RepoPath, os.ModePerm) // Make sure directory exists

	w3 := &lumberjack.Logger{
		Filename:   path.Join(config.RepoPath, "logs", "bitcoin.log"),
		MaxSize:    10, // Megabytes
		MaxBackups: 3,
		MaxAge:     30, // Days
	}
	bitcoinFile := logging.NewLogBackend(w3, "", 0)
	bitcoinFileFormatter := logging.NewBackendFormatter(bitcoinFile, fileLogFormat)
	ml := logging.MultiLogger(bitcoinFileFormatter)
	config.Logger = ml

	walletdb, err := db.Create(config.RepoPath)
	if err != nil {
		return nil, err
	}
	config.DB = walletdb
	mnemonic, err := walletdb.GetMnemonic()
	if err != nil {
		b := make([]byte, 32)
		rand.Read(b)
		mn, err := bip39.NewMnemonic(b)
		if err != nil {
			return nil, err
		}
		err = walletdb.SetMnemonic(mn)
		if err != nil {
			return nil, err
		}
		err = walletdb.SetCreationDate(time.Now())
		if err != nil {
			return nil, err
		}
		mnemonic = mn
	}
	config.Mnemonic = mnemonic

	config.ExchangeRateProvider = NewBitcoinCashPriceFetcher(nil)

	wallet, err := bitcoincash.NewSPVWallet(config)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

func GetRepoPath() (string, error) {
	// Set default base path and directory name
	path := "~"
	directoryName := "ipfsindex"

	// Override OS-specific names
	switch runtime.GOOS {
	case "linux":
		directoryName = ".ipfsindex"
	case "darwin":
		path = "~/Library/Application Support"
	}

	// Join the path and directory name, then expand the home path
	fullPath, err := homedir.Expand(filepath.Join(path, directoryName))
	if err != nil {
		return "", err
	}

	// Return the shortest lexical representation of the path
	return filepath.Clean(fullPath), nil
}
