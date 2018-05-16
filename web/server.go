package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/cpacia/BitcoinCash-Wallet"
	"github.com/cpacia/ipfsindex/app"
	"github.com/cpacia/ipfsindex/db"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
	"gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var log = logging.MustGetLogger("web")

type Server struct {
	ctx            context.Context
	wallet         *bitcoincash.SPVWallet
	router         *mux.Router
	fileServer     http.Handler
	etagFactory    *EtagFactory
	port           int
	listener       *app.TransactionListener
	db             *db.Database
	siteData       *SiteData
	addrChan       chan [2]string
	disconnectChan chan string
	openSockets    map[string]*websocket.Conn
	socketLock     sync.RWMutex
}

type SiteData struct {
	Title         string
	AddressPrefix string
	Hostname      string
	Port          int
}

type Config struct {
	Wallet   *bitcoincash.SPVWallet
	Listener *app.TransactionListener
	Db       *db.Database

	Hostname string
	Port     int

	AddrChan chan [2]string
}

func NewServer(conf Config) (*Server, error) {
	router := mux.NewRouter()
	ef, err := NewEtagFactory("./web/static/")
	if err != nil {
		return nil, err
	}
	var addrPrefix = "bitcoincash:"
	if conf.Wallet.Params().Name == chaincfg.TestNet3Params.Name {
		addrPrefix = "bchtest:"
	} else if conf.Wallet.Params().Name == chaincfg.RegressionNetParams.Name {
		addrPrefix = "bchreg:"
	}

	s := &Server{
		ctx:         context.Background(),
		wallet:      conf.Wallet,
		fileServer:  http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))),
		etagFactory: ef,
		port:        conf.Port,
		listener:    conf.Listener,
		router:      router,
		db:          conf.Db,
		siteData: &SiteData{
			Title:         "Decentralized File Index for IPFS",
			AddressPrefix: addrPrefix,
			Hostname:      conf.Hostname,
			Port:          conf.Port,
		},
		addrChan:       conf.AddrChan,
		disconnectChan: make(chan string),
		openSockets:    make(map[string]*websocket.Conn),
		socketLock:     sync.RWMutex{},
	}
	router.PathPrefix("/static").Methods("GET").Handler(http.HandlerFunc(s.serveFiles))
	router.PathPrefix("/file").Methods("GET").Handler(http.HandlerFunc(s.renderDetails))
	router.HandleFunc("/addfile", s.submitAddFile).Methods("POST")
	router.HandleFunc("/validatecid", s.submitValidateCid).Methods("POST")
	router.HandleFunc("/vote", s.submitVote).Methods("POST")
	router.HandleFunc("/", s.renderIndex).Methods("GET")
	router.HandleFunc("/ws", s.handleWebsocket)
	go s.ProcessSocketRequests()
	return s, nil
}

func (s *Server) Start() {
	go s.wallet.Start()
	http.ListenAndServe(":"+strconv.Itoa(s.port), s.router)
}

func (s *Server) Stop() {
	_, cancel := context.WithCancel(s.ctx)
	cancel()
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
	templates, err := template.ParseFiles(path.Join("web", "templates", "index.html"), path.Join("web", "templates", "header.html"), path.Join("web", "templates", "footer.html"))
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	templates.Lookup("header").ExecuteTemplate(w, "header", s.siteData)
	templates.Lookup("index").ExecuteTemplate(w, "index", nil)
	templates.Lookup("footer").ExecuteTemplate(w, "footer", nil)
}

func (s *Server) renderDetails(w http.ResponseWriter, r *http.Request) {
	pth := strings.Split(r.URL.Path, "/")
	if len(pth) < 3 {
		// TODO: redirect to a fail whale
		w.WriteHeader(http.StatusNotFound)
		return
	}
	txid := pth[2]
	fd := new(db.FileDescriptor)
	if s.db.Where("txid = ?", txid).First(fd).RecordNotFound() {
		// TODO: redirect to a fail whale
		w.WriteHeader(http.StatusNotFound)
		return
	}
	templates, err := template.ParseFiles(path.Join("web", "templates", "details.html"), path.Join("web", "templates", "header.html"), path.Join("web", "templates", "footer.html"))
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	type Details struct {
		Description   string
		Cid           string
		Timestamp     string
		Txid          string
		Category      string
		Upvotes       int64
		Downvotes     int64
		Confirmations uint32
	}
	confirms := uint32(0)
	height, _ := s.wallet.ChainTip()
	if fd.Height > 0 {
		confirms = (height - fd.Height) + 1
	}
	det := Details{
		Description: fd.Description,
		Cid: fd.Cid,
		Timestamp: fd.Timestamp.Format("Mon Jan 2 15:04:05 MST 2006"),
		Txid: fd.Txid,
		Category: fd.Category,
		Upvotes: fd.Upvotes,
		Confirmations: confirms,
	}
	templates.Lookup("header").ExecuteTemplate(w, "header", s.siteData)
	templates.Lookup("details").ExecuteTemplate(w, "details", &det)
	templates.Lookup("footer").ExecuteTemplate(w, "footer", nil)
}

func (s *Server) submitAddFile(w http.ResponseWriter, r *http.Request) {
	type AddFile struct {
		Cid         string `json:"cid"`
		Description string `json:"description"`
		Category    string `json:"category"`
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
		Script:      &app.AddFileScript{*id, af.Description, af.Category},
		Timestamp:   time.Now(),
		Address:     addr,
		AmountToPay: amount,
	}
	if _, err := entry.Script.Serialize(); err == app.ErrInvalidLength {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.listener.NewEntry(addr, entry)
	fmt.Fprintf(w, `{"paymentAddress": "%s", "amountToPay": %f}`, addr.String(), btcutil.Amount(amount).ToBTC())
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
	if _, err := entry.Script.Serialize(); err == app.ErrInvalidLength {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.listener.NewEntry(addr, entry)
	fmt.Fprintf(w, `{"paymentAddress": "%s", "amountToPay": %f}`, addr.String(), btcutil.Amount(amount).ToBTC())
	//TODO: map websocket
}

func (s *Server) submitValidateCid(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Cid string `json:"cid"`
	}
	req := new(Req)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id, err := cid.Decode(req.Cid)
	if err != nil {
		fmt.Fprint(w, `{"valid": false}`)
	} else {
		fmt.Fprintf(w, `{"valid": true, "length": %d}`, len(id.Bytes()))
	}
}
