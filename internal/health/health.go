package health

import (
	"sync/atomic"
	"time"
)

type Stats struct {
	StartTime         time.Time `json:"start_time"`
	RequestsTotal     uint64    `json:"requests_total"`
	FailoversTotal    uint64    `json:"failovers_total"`
	LocalRoutedTotal  uint64    `json:"local_routed_total"`
	CloudRoutedTotal  uint64    `json:"cloud_routed_total"`
	StreamingTotal    uint64    `json:"streaming_total"`
	NonStreamingTotal uint64    `json:"non_streaming_total"`
}

type Tracker struct {
	start time.Time
	req   atomic.Uint64
	fo    atomic.Uint64
	loc   atomic.Uint64
	cld   atomic.Uint64
	str   atomic.Uint64
	non   atomic.Uint64
}

func NewTracker() *Tracker          { return &Tracker{start: time.Now()} }
func (t *Tracker) IncRequest()      { t.req.Add(1) }
func (t *Tracker) IncFailover()     { t.fo.Add(1) }
func (t *Tracker) IncLocal()        { t.loc.Add(1) }
func (t *Tracker) IncCloud()        { t.cld.Add(1) }
func (t *Tracker) IncStreaming()    { t.str.Add(1) }
func (t *Tracker) IncNonStreaming() { t.non.Add(1) }

func (t *Tracker) Snapshot() Stats {
	return Stats{StartTime: t.start, RequestsTotal: t.req.Load(), FailoversTotal: t.fo.Load(), LocalRoutedTotal: t.loc.Load(), CloudRoutedTotal: t.cld.Load(), StreamingTotal: t.str.Load(), NonStreamingTotal: t.non.Load()}
}
