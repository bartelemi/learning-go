package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
)

type PlayerStore interface {
	GetPlayerScore(name string) int
	RecordWin(name string)
}

type VolatilePlayerStore struct {
	scores map[string]int
}

func (v *VolatilePlayerStore) GetPlayerScore(name string) int {
	return v.scores[name]
}

func (v *VolatilePlayerStore) RecordWin(name string) {
	v.scores[name]++
}

func NewVolatilePlayerStore() *VolatilePlayerStore {
	return &VolatilePlayerStore{map[string]int{}}
}

const (
	ScoresBucket = "Scores"
)

type BoltPlayerStore struct {
	db *bolt.DB
}

func (b *BoltPlayerStore) GetPlayerScore(name string) int {
	score := 0
	err := b.db.View(func (tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ScoresBucket))
		v := b.Get([]byte(name))
		if v != nil {
			score = int(binary.BigEndian.Uint32(v))
		}
		return nil
	})
	if err != nil {
		fmt.Errorf("get player score: %s", err)
	}
	return score
}

func (b *BoltPlayerStore) RecordWin(name string) {
	err := b.db.Update(func (tx *bolt.Tx) error {
		score := uint32(b.GetPlayerScore(name) + 1)
		scoreBuffer := make([]byte, 4)
		binary.BigEndian.PutUint32(scoreBuffer, score)
		b := tx.Bucket([]byte(ScoresBucket))
		err := b.Put([]byte(name), scoreBuffer)
		return err
	})
	if err != nil {
		fmt.Errorf("record win: %s", err)
	}
}

func (b *BoltPlayerStore) Close() {
	b.db.Close()
}

func NewBoltPlayerStore(path string) (store *BoltPlayerStore, err error) {
	db, err := bolt.Open(
		path,
		0600,
		&bolt.Options{Timeout: 1 * time.Second},
	)
	if err != nil {
		return nil, err
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(ScoresBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)		
		}
		return nil
	})
	return &BoltPlayerStore{db}, nil
}

type PlayerServer struct {
	store PlayerStore
}

func (p *PlayerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	player := r.URL.Path[len("/players/"):]
	switch r.Method {
	case http.MethodGet:
		p.showScore(w, player)
	case http.MethodPost:
		p.processWin(w, player)
	}
}

func (p *PlayerServer) showScore(w http.ResponseWriter, name string) {
	score := p.store.GetPlayerScore(name)
	if score == 0 {
		w.WriteHeader(http.StatusNotFound)
	}
	fmt.Fprint(w, score)
}

func (p *PlayerServer) processWin(w http.ResponseWriter, name string) {
	p.store.RecordWin(name)
	w.WriteHeader(http.StatusAccepted)
}

func GetPlayerScore(name string) int {
	switch name {
	case "Floyd":
		return 10
	case "Pepper":
		return 20
	default:
		return 0
	}
}
