package handler

import (
	"log"

	"github.com/syndtr/goleveldb/leveldb"
)

type Storage struct {
	path string
	db   *leveldb.DB
}
type StorageEngine interface {
	Write(key, val string) error
	Read(key string) (string, error)
	Delete(key string) error
}

func NewStorage(path string) StorageEngine {
	var storageEngine StorageEngine

	s := &Storage{
		path: path,
	}

	//初始化存储引擎
	s.initdb()
	storageEngine = s

	return storageEngine
}

//initdb 初始化存储.
func (s *Storage) initdb() {
	db, err := leveldb.OpenFile(s.path, nil)
	if err != nil {
		panic(err)
	}
	s.db = db
}

//Read 读取操作.
func (s *Storage) Read(key string) (str string, err error) {
	val, err := s.db.Get([]byte(key), nil)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return str, err
	}

	return string(val), nil
}

//Write 写操作.
func (s *Storage) Write(key, value string) error {
	err := s.db.Put([]byte(key), []byte(value), nil)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return err
	}

	return nil
}

//Delete 删除.
func (s *Storage) Delete(key string) error {
	err := s.db.Delete([]byte(key), nil)
	if err != nil {
		log.Printf("err:%+v\n", err)
		return err
	}

	return nil
}
