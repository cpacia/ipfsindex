package db

import (
	"github.com/blevesearch/bleve"
	"github.com/jinzhu/gorm"
	"path"
	"time"
)

type FileDescriptor struct {
	gorm.Model
	Txid        string    `json:"txid" gorm:"index;unique;not null"`
	Cid         string    `json:"cid"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Timestamp   time.Time `json:"timestamp"`
	Upvotes     int64     `json:"upvotes"`
	Downvotes   int64     `json:"downvotes"`
	Net         int64     `json:"net"`
	Height      uint32    `json:"height"`
}

type Vote struct {
	gorm.Model
	FDTxid    string    `json:"fdTxid" gorm:"index;not null"`
	Txid      string    `json:"txid" gorm:"unique;not null"`
	Comment   string    `json:"comment"`
	Timestamp time.Time `json:"timestamp"`
	Upvote    bool      `json:"upvote"`
	Height    uint32    `json:"height"`
}

type Database struct {
	*gorm.DB
	search bleve.Index
}

func NewDatabase(repoPath string) (*Database, error) {
	db, err := gorm.Open("sqlite3", path.Join(repoPath, "the_index.db"))
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&FileDescriptor{}, &Vote{})

	index, err := bleve.Open(path.Join(repoPath, "index.bleve"))
	if err == bleve.ErrorIndexPathDoesNotExist {
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(path.Join(repoPath, "index.bleve"), mapping)
		if err != nil {
			return nil, err
		}
	}
	database := &Database{
		DB:     db,
		search: index,
	}
	return database, nil
}

func (db *Database) Index(txid string, fd FileDescriptor) {
	db.search.Index(txid, fd)
}

func (db *Database) Query(searchTerm string, limit int, offset int) ([]string, error) {
	var ids []string
	query := bleve.NewMatchQuery(searchTerm)
	search := bleve.NewSearchRequest(query)
	search.Size = limit
	search.From = offset
	searchResults, err := db.search.Search(search)
	if err != nil {
		return ids, err
	}
	for _, r := range searchResults.Hits {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

func (db *Database) Close() {
	db.Close()
}
