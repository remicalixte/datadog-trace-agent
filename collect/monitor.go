package collect

import (
	"time"

	"github.com/DataDog/datadog-trace-agent/statsd"
)

// monitor runs a loop, sending occasional statsd entries.
func (c *Cache) monitor(client statsd.StatsClient) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			c.computeStats(client)
		}
	}
}

func (c *Cache) computeStats(client statsd.StatsClient) {
	iter := c.newReverseIterator()
	now := time.Now()
	var (
		n2m    float64
		n5m    float64
		nLarge float64
	)
	for {
		t, ok := iter.getAndAdvance()
		if !ok {
			break
		}
		idle := now.Sub(t.lastmod)
		switch {
		case idle <= 30*time.Second:
			break // stop counting, these are recent spans
		case idle < 2*time.Minute:
			n2m++
		case idle < 5*time.Minute:
			n5m++
		default:
			nLarge++
		}
	}

	total := float64(iter.len())
	client.Gauge("datadog.trace_agent.cache.traces", total-n2m-n5m-nLarge, []string{
		"version:v1",
		"idle:under30sec",
	}, 1)

	client.Gauge("datadog.trace_agent.cache.traces", n2m, []string{
		"version:v1",
		"idle:under2m",
	}, 1)

	client.Gauge("datadog.trace_agent.cache.traces", n5m, []string{
		"version:v1",
		"idle:under5m",
	}, 1)

	client.Gauge("datadog.trace_agent.cache.traces", nLarge, []string{
		"version:v1",
		"idle:over5m",
	}, 1)

	c.mu.RLock()
	client.Gauge("datadog.trace_agent.cache.bytes", float64(c.size), []string{"version:v1"}, 1)
	c.mu.RUnlock()
}
