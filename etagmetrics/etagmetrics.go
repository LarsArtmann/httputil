package etagmetrics

import (
	"sync/atomic"

	etag "github.com/larsartmann/go-etag/server"
)

// Counters holds the atomic event counters for one etag middleware
// configuration. Create one via [Attach]; all methods are safe for
// concurrent use.
type Counters struct {
	// Generated counts OnETagGenerated events: one per response whose ETag
	// the middleware computed and set. It fires for 200 responses and also
	// for 304s (the tag is resolved to answer the conditional), but not for
	// streamed responses that exceeded MaxBufferSize.
	Generated atomic.Int64
	// NotModified counts On304 events: one per conditional request answered
	// with 304 Not Modified.
	NotModified atomic.Int64
	// BufferOverflows counts OnBufferOverflow events: one per response whose
	// body exceeded MaxBufferSize and was switched to streaming (the partial
	// buffer still gets a tag at flush time).
	BufferOverflows atomic.Int64
}

// Snapshot is a point-in-time copy of [Counters] values.
type Snapshot struct {
	Generated       int64
	NotModified     int64
	BufferOverflows int64
}

// Attach installs counting hooks on a copy of cfg and returns the modified
// configuration (pass it to etag.New) together with the counters it will
// update. Hooks already present on cfg are preserved and run after the
// counting hooks, so attaching never silently drops consumer instrumentation.
func Attach(cfg etag.ETagConfig) (etag.ETagConfig, *Counters) {
	c := &Counters{}

	generated := cfg.OnETagGenerated
	cfg.OnETagGenerated = func(e etag.ETag) {
		c.Generated.Add(1)
		if generated != nil {
			generated(e)
		}
	}

	notModified := cfg.On304
	cfg.On304 = func(e etag.ETag) {
		c.NotModified.Add(1)
		if notModified != nil {
			notModified(e)
		}
	}

	overflow := cfg.OnBufferOverflow
	cfg.OnBufferOverflow = func(limit int) {
		c.BufferOverflows.Add(1)
		if overflow != nil {
			overflow(limit)
		}
	}

	return cfg, c
}

// Snapshot returns the current counter values.
func (c *Counters) Snapshot() Snapshot {
	return Snapshot{
		Generated:       c.Generated.Load(),
		NotModified:     c.NotModified.Load(),
		BufferOverflows: c.BufferOverflows.Load(),
	}
}

// HitRatio returns the fraction of tag-computing responses answered with
// 304 Not Modified: NotModified / (Generated + NotModified). It returns 0
// before any event fires. Both hooks fire for a 304 (the tag is resolved to
// answer the conditional), so the denominator counts every tag-computing
// response exactly once.
func (c *Counters) HitRatio() float64 {
	gen := c.Generated.Load()
	nm := c.NotModified.Load()
	total := gen + nm
	if total == 0 {
		return 0
	}

	return float64(nm) / float64(total)
}
