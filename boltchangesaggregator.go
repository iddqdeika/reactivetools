package reactivetools

import (
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	bolt "go.etcd.io/bbolt"
	"sync"
)

var (
	bucketName = []byte("main")
)

func NewBoltChangesAggregator(cfg helpful.Config, l helpful.Logger) (ChangesAggregator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil Config")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil Logger")
	}

	storagePath, err := cfg.GetString("bolt_storage_path")
	if err != nil {
		return nil, err
	}
	db, err := bolt.Open(storagePath, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &boltChangesAggregator{
		db: db,
	}, nil
}

type boltChangesAggregator struct {
	db *bolt.DB
	m  sync.RWMutex
}

func (b *boltChangesAggregator) Set(key string, val string) error {
	b.m.Lock()
	defer b.m.Unlock()
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return fmt.Errorf("bucket " + string(bucketName) + " does not exist, might not be initialized")
		}
		return bucket.Put([]byte(key), []byte(val))
	})
}

func (b *boltChangesAggregator) Get(key string) (string, error) {
	b.m.RLock()
	defer b.m.RUnlock()
	var result string
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return fmt.Errorf("bucket " + string(bucketName) + " does not exist, might not be initialized")
		}
		data := bucket.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("value for key " + key + "does not exist in bucket " + string(bucketName))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

func (b *boltChangesAggregator) Process(event ChangeEvent) error {
	if event == nil {
		return fmt.Errorf("given event is nil")
	}
	return b.Set(event.ObjectIdentifier(), event.Data())
}

func (b *boltChangesAggregator) Close() error {
	return b.db.Close()
}
