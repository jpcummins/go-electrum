package electrum

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func mustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b[:])
}

type TxCache struct {
	db *sql.DB
}

func NewTxCache(db *sql.DB) (*TxCache, error) {
	if db == nil {
		var err error
		db, err = sql.Open("sqlite3", "tx_cache.db")
		if err != nil {
			return nil, err
		}
	}

	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS tx_cache (
		txid VARCHAR(64) PRIMARY KEY,
		tx TEXT,
		is_detailed INTEGER DEFAULT 0
	)
	`)
	if err != nil {
		return nil, err
	}
	return &TxCache{db}, nil
}

func (c *TxCache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *TxCache) Store(txID string, tx any) error {
	b, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	isDetailed := 0
	switch tx.(type) {
	case *DetailedTransaction:
		isDetailed = 1
	case DetailedTransaction:
		isDetailed = 1
	}

	_, err = c.db.Exec(
		`INSERT INTO tx_cache (txid, tx, is_detailed) VALUES (?, ?, ?)
		ON CONFLICT(txid) WHERE is_detailed = 0 DO UPDATE SET
			tx = ?,
			is_detailed = ?`,
		txID,
		string(b[:]),
		isDetailed,
		string(b[:]),
		isDetailed,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *TxCache) Load(txID string, tx any) (ok bool) {
	row, err := c.db.Query(
		"SELECT tx, is_detailed FROM tx_cache WHERE txid = ?",
		txID,
	)
	if err != nil {
		return false
	}
	defer row.Close()
	if !row.Next() {
		return false
	}
	var data []byte
	var isDetailed int
	err = row.Scan(&data, &isDetailed)
	if err != nil {
		return false
	}

	switch tx.(type) {
	case *DetailedTransaction:
		if isDetailed == 0 {
			return false
		}
	}
	err = json.Unmarshal(data, tx)
	if err != nil {
		return false
	}
	return true
}
