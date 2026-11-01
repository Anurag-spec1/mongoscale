package metrics

import (
	"sync/atomic"
	"time"
)

type MongoOps struct {
	slowThreshold time.Duration
	reads         atomic.Int64
	writes        atomic.Int64
	slow          atomic.Int64
	failed        atomic.Int64
}

func NewMongoOps(slowThreshold time.Duration) *MongoOps {
	return &MongoOps{slowThreshold: slowThreshold}
}

func (m *MongoOps) Record(command string, duration time.Duration, failed bool) {
	switch command {
	case "find", "aggregate", "count", "distinct", "getMore":
		m.reads.Add(1)
	case "insert", "update", "delete", "findAndModify":
		m.writes.Add(1)
	default:
		return
	}
	if failed {
		m.failed.Add(1)
	}
	if duration >= m.slowThreshold {
		m.slow.Add(1)
	}
}

func (m *MongoOps) Reads() int64 {
	return m.reads.Load()
}

func (m *MongoOps) Writes() int64 {
	return m.writes.Load()
}

func (m *MongoOps) Slow() int64 {
	return m.slow.Load()
}

func (m *MongoOps) Failed() int64 {
	return m.failed.Load()
}
