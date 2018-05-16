package main

import (
	"errors"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/cpacia/ipfsindex/app"
	"github.com/cpacia/ipfsindex/db"
	"github.com/cpacia/ipfsindex/web"
	"github.com/jessevdk/go-flags"
	"github.com/op/go-logging"
	"net"
	"os"
	"os/signal"
)

var parser = flags.NewParser(nil, flags.Default)

type Start struct {
	Testnet     bool   `short:"t" long:"testnet" description:"use the test network"`
	Regtest     bool   `short:"r" long:"regtest" description:"run in regression test mode"`
	Port        int    `short:"p" long:"port" description:"the web server port" default:"8080"`
	Hostname    string `short:"h" long:"hostname" description:"the hostname for the server" default:"localhost"`
	TrustedPeer string `short:"i" long:"trustedpeer" description:"specify a single trusted peer to connect to"`
}

var stdoutLogFormat = logging.MustStringFormatter(
	`%{color:reset}%{color}%{time:15:04:05.000} [%{shortfunc}] [%{level}] %{message}`,
)

var log = logging.MustGetLogger("main")

var start Start

var server *web.Server

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if server != nil {
				server.Stop()
			}
			os.Exit(1)
		}
	}()
	parser.AddCommand("start",
		"start the server",
		"The start command starts the web server and wallet",
		&start)
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}

func (x *Start) Execute(args []string) error {
	backendStdout := logging.NewLogBackend(os.Stdout, "", 0)
	backendStdoutFormatter := logging.NewBackendFormatter(backendStdout, stdoutLogFormat)
	logging.SetBackend(backendStdoutFormatter)

	if x.Testnet && x.Regtest {
		return errors.New("Invalid combination of testnet and regtest")
	}
	var trustedPeer net.Addr
	var err error
	params := &chaincfg.MainNetParams
	if x.Testnet {
		params = &chaincfg.TestNet3Params
	} else if x.Regtest {
		if x.TrustedPeer == "" {
			return errors.New("Must specify a  trusted peer if using regtest")
		}
		params = &chaincfg.RegressionNetParams
		trustedPeer, err = net.ResolveTCPAddr("ip4", x.TrustedPeer)
		if err != nil {
			return err
		}
	}
	repoPath, err := app.GetRepoPath()
	if err != nil {
		return err
	}

	database, err := db.NewDatabase(repoPath)
	if err != nil {
		return err
	}

	wallet, err := app.NewWallet(params, repoPath, trustedPeer)
	if err != nil {
		return err
	}

	addrChan := make(chan [2]string)
	tl := app.NewTransactionListener(wallet, database, addrChan)
	wallet.AddTransactionListener(tl.ListenBitcoinCash)

	conf := web.Config{
		Wallet:   wallet,
		Listener: tl,
		Db:       database,
		Port:     x.Port,
		Hostname: x.Hostname,
		AddrChan: addrChan,
	}

	webServer, err := web.NewServer(conf)
	if err != nil {
		return err
	}
	webServer.Start()
	return nil
}
