package db

import (
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
}

func NewDatabase(repoPath string) (*Database, error) {
	db, err := gorm.Open("sqlite3", path.Join(repoPath, "the_index.db"))
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&FileDescriptor{}, &Vote{})
	return &Database{db}, nil
}

func (db *Database) Close() {
	db.Close()
}
