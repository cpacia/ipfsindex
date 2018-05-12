package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/cpacia/BitcoinCash-Wallet"
	"github.com/cpacia/ipfsindex/app"
	"github.com/gorilla/mux"
	"github.com/op/go-logging"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/cpacia/ipfsindex/db"
)

var log = logging.MustGetLogger("web")

type Server struct {
	wallet      *bitcoincash.SPVWallet
	router      *mux.Router
	fileServer  http.Handler
	etagFactory *EtagFactory
	port        int
	listener    *app.TransactionListener
	db          *db.Database
}

func NewServer(wallet *bitcoincash.SPVWallet, listener *app.TransactionListener, db *db.Database, port int) (*Server, error) {
	router := mux.NewRouter()
	ef, err := NewEtagFactory("./web/static/")
	if err != nil {
		return nil, err
	}
	s := &Server{
		wallet:      wallet,
		fileServer:  http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))),
		etagFactory: ef,
		port:        port,
		listener:    listener,
		router:      router,
		db:          db,
	}
	router.PathPrefix("/static").Methods("GET").Handler(http.HandlerFunc(s.serveFiles))
	router.HandleFunc("/addfile", s.submitAddFile).Methods("POST")
	router.HandleFunc("/vote", s.submitVote).Methods("POST")
	router.HandleFunc("/", s.renderIndex).Methods("GET")
	return s, nil
}

func (s *Server) Start() {
	go s.wallet.Start()
	http.ListenAndServe(":"+strconv.Itoa(s.port), s.router)
}

func (s *Server) Stop() {
	s.db.Close()
	s.wallet.Close()
}

func (s *Server) serveFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=86400")
	e, err := s.etagFactory.GetEtag(r.URL.Path)
	if err == nil {
		w.Header().Set("Etag", e)
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, e) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}
	s.fileServer.ServeHTTP(w, r)
}

func (s *Server) renderIndex(w http.ResponseWriter, r *http.Request) {

}

func (s *Server) submitAddFile(w http.ResponseWriter, r *http.Request) {
	type AddFile struct {
		Cid         string `json:"cid"`
		Description string `json:"description"`
	}
	af := new(AddFile)
	err := json.NewDecoder(r.Body).Decode(af)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := cid.Decode(af.Cid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	amount, err := app.MinimumInputSize(s.wallet)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	addr := s.wallet.CurrentAddress(wallet.EXTERNAL)
	b := make([]byte, 20)
	rand.Read(b)
	entry := app.UserEntry{
		ID:          hex.EncodeToString(b),
		Script:      &app.AddFileScript{*id, af.Description},
		Timestamp:   time.Now(),
		Address:     addr,
		AmountToPay: amount,
	}
	s.listener.NewEntry(addr, entry)
	fmt.Fprintf(w, `{"paymentAddress": "%s", "amountToPay": %d}`, addr.String(), amount)
	//TODO: map websocket
}

func (s *Server) submitVote(w http.ResponseWriter, r *http.Request) {
	type Vote struct {
		Txid        string `json:"txid"`
		Upvote      bool   `json:"upvote"`
		Description string `json:"comment"`
	}
	v := new(Vote)
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fd := &db.FileDescriptor{}
	if s.db.Where("txid = ?", v.Txid).First(fd).RecordNotFound() || fd.Height <= 0 {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Please wait for the file transaction to confirm before voting")
		return
	}

	txid, err := chainhash.NewHashFromStr(v.Txid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	amount, err := app.MinimumInputSize(s.wallet)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	addr := s.wallet.CurrentAddress(wallet.EXTERNAL)
	b := make([]byte, 20)
	rand.Read(b)
	entry := app.UserEntry{
		ID:          hex.EncodeToString(b),
		Script:      &app.VoteScript{*txid, v.Description, v.Upvote},
		Timestamp:   time.Now(),
		Address:     addr,
		AmountToPay: amount,
	}
	s.listener.NewEntry(addr, entry)
	fmt.Fprintf(w, `{"paymentAddress": "%s", "amountToPay": %d}`, addr.String(), amount)
	//TODO: map websocket
}