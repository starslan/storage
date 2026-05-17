package wal

import "strconv"

type Record struct {
	id     int64
	data   string
	doneCh chan error
}

func (record *Record) String() string {
	return strconv.FormatInt(record.id, 10) + "_" + record.data
}

func NewRecord(data string, id int64) *Record {
	return &Record{
		id:     id,
		data:   data,
		doneCh: make(chan error, 1),
	}
}
