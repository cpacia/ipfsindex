package web

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

var EtagNotFoundError = errors.New("etag not found")

type EtagFactory struct {
	filemap map[string]string
}

func NewEtagFactory(staticFileDir string) (*EtagFactory, error) {
	ef := &EtagFactory{make(map[string]string)}
	err := filepath.Walk(staticFileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			hash := sha256.Sum256(content)
			ef.filemap["/"+path] = `"` + base64.StdEncoding.EncodeToString(hash[:]) + `"`
		}
		return nil
	})
	return ef, err
}

func (ef *EtagFactory) GetEtag(filePath string) (string, error) {
	etag, ok := ef.filemap[filePath]
	if !ok {
		return "", EtagNotFoundError
	}
	return etag, nil
}
