package storage

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type BpRecord struct {
	Ts    time.Time
	Sys   byte
	Dia   byte
	Pulse byte
}

var (
	db              *badger.DB
	records         []*BpRecord
	mutex           = sync.RWMutex{}
	dbPrefixRecords = []byte("bpnRecord:")
)

func Connect(path string) error {
	badgerDb, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return err
	}
	db = badgerDb
	if err := InitRecords(); err != nil {
		return err
	}
	return nil
}

func Close() error {
	return db.Close()
}

func InitRecords() error {
	return db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		mutex.Lock()
		defer mutex.Unlock()

		records = nil
		for it.Seek(dbPrefixRecords); it.ValidForPrefix(dbPrefixRecords); it.Next() {
			item := it.Item()
			record := &BpRecord{}
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, record)
			}); err != nil {
				return fmt.Errorf("cannot read value: %v", err)
			}
			records = append(records, record)
		}

		return nil
	})
}

func Read() []*BpRecord {
	mutex.RLock()
	defer mutex.RUnlock()
	return records
}

func Save(rec *BpRecord) error {
	mutex.Lock()
	defer mutex.Unlock()

	timestamp := strconv.FormatInt(rec.Ts.Unix(), 10)
	key := slices.Concat(dbPrefixRecords, []byte(timestamp))

	jsonRecord, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("cannot serialize json: %v", err)
	}

	if err := db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsonRecord)
	}); err != nil {
		return fmt.Errorf("failed to store record: %w", err)
	}

	records = append(records, rec)
	slices.SortFunc(records, func(a, b *BpRecord) int {
		return a.Ts.Compare(b.Ts)
	})

	return nil
}

func (r BpRecord) String() string {
	return fmt.Sprintf("%s - Sys: %d, Dia: %d, Pulse: %d", r.Ts.UTC().Format("2006-01-02 15:04:05"), r.Sys, r.Dia, r.Pulse)
}
