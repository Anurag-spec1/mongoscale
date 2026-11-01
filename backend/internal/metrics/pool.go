package metrics

import (
	"sync/atomic"

	"go.mongodb.org/mongo-driver/event"

	"mongoscale/internal/models"
)

type PoolStats struct {
	maxSize        int64
	created        atomic.Int64
	closed         atomic.Int64
	checkoutStart  atomic.Int64
	checkedOut     atomic.Int64
	checkedIn      atomic.Int64
	checkoutFailed atomic.Int64
}

func NewPoolStats(maxSize uint64) *PoolStats {
	return &PoolStats{maxSize: int64(maxSize)}
}

func (p *PoolStats) Handle(poolEvent *event.PoolEvent) {
	switch poolEvent.Type {
	case "ConnectionCreated":
		p.created.Add(1)
	case "ConnectionClosed":
		p.closed.Add(1)
	case "ConnectionCheckOutStarted":
		p.checkoutStart.Add(1)
	case "ConnectionCheckedOut":
		p.checkedOut.Add(1)
	case "ConnectionCheckedIn":
		p.checkedIn.Add(1)
	case "ConnectionCheckOutFailed":
		p.checkoutFailed.Add(1)
	}
}

func (p *PoolStats) Usage() models.PoolUsage {
	inUse := p.checkedOut.Load() - p.checkedIn.Load()
	waiting := p.checkoutStart.Load() - p.checkedOut.Load() - p.checkoutFailed.Load()
	return models.PoolUsage{
		Open:    max(p.created.Load()-p.closed.Load(), 0),
		InUse:   max(inUse, 0),
		Waiting: max(waiting, 0),
		Max:     p.maxSize,
		Failed:  p.checkoutFailed.Load(),
	}
}
