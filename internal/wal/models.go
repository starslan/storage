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

func NewRecord(data string) *Record {
	return &Record{
		data:   data,
		doneCh: make(chan error, 1),
	}
}
