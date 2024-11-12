package botauth

import (
	"github.com/rosedblabs/rosedb/v2"
)

type Cache interface {
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

type RoseCache struct {
	db *rosedb.DB
}

func NewRoseCache(path string) *RoseCache {
	options := rosedb.DefaultOptions
	options.DirPath = path
	db, err := rosedb.Open(options)
	if err != nil {
		panic(err)
	}
	return &RoseCache{db: db}
}

func (c *RoseCache) Get(key []byte) ([]byte, error) {
	return c.db.Get(key)
}

func (c *RoseCache) Put(key []byte, value []byte) error {
	return c.db.Put(key, value)
}

func (c *RoseCache) Delete(key []byte) error {
	return c.db.Delete(key)
}
