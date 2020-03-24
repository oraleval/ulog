package filter

import (
	"github.com/ouqiang/timewheel"
	"github.com/tidwall/gjson"
	"github.com/xujiajun/nutsdb"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var (
	logBlack = sync.Map{}
	DBBucket = "logBucket"
)

type LogCacheData struct {
	IsError bool
}

type LogFilter struct {
	W  io.Writer
	DB *nutsdb.DB
	TW *timewheel.TimeWheel
}

func NewLogFilter(w io.Writer, dir string) *LogFilter {
	l := &LogFilter{W: w}

	_ = os.RemoveAll(dir)
	opt := nutsdb.DefaultOptions
	opt.Dir = dir
	db, err := nutsdb.Open(opt)
	if err != nil {
		log.Printf("new nutsdb err:%v\n", err)
		return l
	}

	l.TW = LogCacheReload(l)
	l.DB = db
	l.TW.Start()

	return l
}

func (f *LogFilter) Write(p []byte) (n int, err error) {
	level := gjson.Get(string(p), "level").String()
	globalID := gjson.Get(string(p), "globalID").String()
	end := gjson.Get(string(p), "end").Bool()
	if globalID != "" {
		globalBool, isGlobalID := logBlack.Load(globalID)
		if end {
			err := f.SendOut(globalID, p, globalBool.(bool))
			if err != nil {
				log.Printf("NewLogFilter SendOut err (%v)\n", err)
				return len(p), err
			}
			return len(p), nil
		}
		if !isGlobalID {
			logBlack.Store(globalID, false)
			f.TW.AddTimer(10*time.Second, globalID, globalID)
		}
		if level == "error" {
			logBlack.Store(globalID, true)
		}
		err := f.AddLogCache(globalID, p)
		if err != nil {
			log.Printf("NewLogFilter AddLogCache err (%v)\n", err)
			return f.W.Write(p)
		}
		return len(p), nil
	}
	if level == "error" {
		return f.W.Write(p)
	}
	return len(p), nil
}

func (f *LogFilter) SendOut(globalID string, p []byte, tmp bool) (err error) {
	logBlack.Delete(globalID)
	time.Sleep(time.Second)
	newList, err := f.GetLogCache(globalID)
	if err != nil {
		log.Printf("GetLogCache globalID (%s) err (%v)\n", globalID, err)
	}
	err = f.DeleteLogCache(globalID)
	if err != nil {
		log.Printf("DeleteLogCache err (%v)\n", err)
	}
	if len(newList) > 0 && tmp {
		for _, n := range newList {
			_, err := f.W.Write(n)
			if err != nil {
				log.Printf("GetLogCache list Write err (%v)\n", err)
			}
		}
		f.W.Write(p)
	}
	f.TW.RemoveTimer(globalID)
	return nil
}

func (f *LogFilter) AddLogCache(globalID string, msg []byte) error {
	err := f.DB.Update(func(tx *nutsdb.Tx) error {
		key := []byte(globalID)
		return tx.SAdd(DBBucket, key, msg)
	})
	if err != nil {
		return err
	}
	return nil
}

func (f *LogFilter) GetLogCache(globalID string) ([][]byte, error) {
	list := make([][]byte, 0)
	err := f.DB.View(func(tx *nutsdb.Tx) error {
		key := []byte(globalID)
		if items, err := tx.SMembers(DBBucket, key); err != nil {
			return err
		} else {
			for _, n := range items {
				list = append(list, n)
			}
		}
		return nil
	})
	return list, err
}

func (f *LogFilter) DeleteLogCache(globalID string) error {
	err := f.DB.Update(func(tx *nutsdb.Tx) error {
		key := []byte(globalID)
		if items, err := tx.SMembers(DBBucket, key); err != nil {
			return err
		} else {
			if len(items) > 0 {
				for _, n := range items {
					err := tx.SRem(DBBucket, key, n)
					if err != nil {
						log.Printf("DeleteLogCache SRem key (%s) err (%v)\n", key, err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func LogCacheReload(l *LogFilter) *timewheel.TimeWheel {
	t := timewheel.New(2*time.Second, 3600, func(data interface{}) {
		logBlack.Delete(data)
		err := l.DeleteLogCache(data.(string))
		if err != nil {
			log.Printf("DeleteLogCache globalID (%v) err (%v)\n", data, err)
		}
	})
	return t
}
