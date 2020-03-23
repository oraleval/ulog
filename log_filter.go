package ulog

import (
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
	"github.com/xujiajun/nutsdb"
	"io"
)

var (
	logCache = map[string][]string{}
	DBBucket = "logBucket"
)

type LogData struct {
	RequestID string `json:"requestID"`
	GlobalID  string `json:"globalID"`
	Level     string `json:"level"`
}

type LogFilter struct {
	W  io.Writer
	DB *nutsdb.DB
}

func NewLogFilter(w io.Writer) *LogFilter {
	l := &LogFilter{W: w}
	opt := nutsdb.DefaultOptions
	opt.Dir = "/tmp/nutsdb"
	db, err := nutsdb.Open(opt)
	defer db.Close()
	if err != nil {
		log.Printf("new nutsdb err:%v\n", err)
		return l
	}
	l.DB = db
	return l
}

func (f *LogFilter) Write(p []byte) (n int, err error) {
	level := gjson.Get(string(p), "level").String()
	globalID := gjson.Get(string(p), "globalID").String()
	if globalID != "" {
		if level == "error" {
			if _, ok := logCache[globalID]; ok && len(logCache[globalID]) > 0 {
				for _, s := range logCache[globalID] {
					f.W.Write([]byte(s))
				}
			}
			logCache[globalID] = []string{}
			return f.W.Write(p)
		} else {
			if _, ok := logCache[globalID]; ok {
				logCache[globalID] = append(logCache[globalID], string(p))
			} else {
				logCache[globalID] = []string{string(p)}
			}
			return len(p), nil
		}
	}
	return f.W.Write(p)
}

func (f *LogFilter) AddLogCache(globalID string, msg []byte) error {
	err := f.DB.Update(func(tx *nutsdb.Tx) error {
		key := []byte(globalID)
		return tx.RPush(DBBucket, key, msg)
	})
	if err != nil {
		return err
	}
	return nil
}
