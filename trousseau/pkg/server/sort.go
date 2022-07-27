package server

import (
	"sort"
	"time"

	"github.com/ondat/trousseau/pkg/logger"
	"k8s.io/klog/v2"
)

const (
	fastestMetricsChanBufferSize = 10
	fastestAverageDiv            = 2
)

var fastestAverageMaxAge = 30 * time.Second

// Roundrobin sorts providers based on Round-robin algorithm.
type Roundrobin struct {
	lock  chan bool
	names []string
	index int
}

// Next returns the next iteration.
func (r *Roundrobin) Next() []string {
	if len(r.names) == 0 {
		return make([]string, 0)
	}

	r.lock <- true
	defer func() {
		<-r.lock
	}()

	r.index++
	if r.index > len(r.names)-1 {
		r.index = 0
	}

	if r.index == 0 {
		klog.V(logger.Debug1).InfoS("Next Round-robin state", "providers", r.names)

		return r.names
	}

	//nolint:gocritic // This is a valid append
	sorted := append(r.names[r.index:], r.names[:r.index]...)

	klog.V(logger.Debug1).InfoS("Next Round-robin state", "providers", sorted)

	return sorted
}

// NewRoundrobin creates a new Roundrobin selector.
func NewRoundrobin(providers []string) *Roundrobin {
	names := append(make([]string, 0, len(providers)), providers...)

	sort.Strings(names)

	return &Roundrobin{
		lock:  make(chan bool, 1),
		names: names,
		index: -1,
	}
}

type avgMetrics struct {
	provider    string
	lastUpdate  time.Time
	reponseTime time.Duration
}

// Metric contains one measurement.
type Metric struct {
	Provider    string
	ReponseTime time.Duration
}

// Fastest sorts providers by response time.
type Fastest struct {
	lock        chan bool
	metricsChan chan Metric
	providers   map[string]*avgMetrics
}

// Fastest returns providers sorted by response time.
func (f *Fastest) Fastest() []string {
	responseTimes := []avgMetrics{}

	f.lock <- true

	for provider := range f.providers {
		if f.providers[provider] == nil || f.providers[provider].lastUpdate.Add(fastestAverageMaxAge).Before(time.Now()) {
			f.providers[provider] = &avgMetrics{
				provider: provider,
			}
		}

		responseTimes = append(responseTimes, *f.providers[provider])
	}

	<-f.lock

	sort.SliceStable(responseTimes, func(i, j int) bool {
		return responseTimes[i].reponseTime.Milliseconds() < responseTimes[j].reponseTime.Milliseconds()
	})

	sorted := []string{}
	for _, rt := range responseTimes {
		sorted = append(sorted, rt.provider)
	}

	klog.V(logger.Debug1).InfoS("Providers names by response time", "providers", sorted)

	return sorted
}

// C returns channel to send metrics.
func (f *Fastest) C() chan<- Metric {
	return f.metricsChan
}

func (f *Fastest) consumeMetrics() {
	klog.Info("Start watching metrics channel...")

	for {
		metrics, ok := <-f.metricsChan
		if !ok {
			klog.Info("Watching metrics channel closed")

			return
		}

		now := time.Now()

		f.lock <- true

		existing, ok := f.providers[metrics.Provider]
		if !ok || existing == nil || existing.lastUpdate.Add(fastestAverageMaxAge).Before(now) {
			existing = &avgMetrics{
				provider: metrics.Provider,
			}
		}

		existing.lastUpdate = now

		if existing.reponseTime == 0 {
			existing.reponseTime = metrics.ReponseTime
		} else {
			existing.reponseTime = (existing.reponseTime + metrics.ReponseTime) / fastestAverageDiv
		}

		f.providers[metrics.Provider] = existing

		<-f.lock

		klog.V(logger.Debug1).InfoS("Current average response time", "provider", metrics.Provider, "ms", existing.reponseTime.Milliseconds())
	}
}

// NewFastest creates a new Fastest selector.
func NewFastest(providers []string) *Fastest {
	fastest := Fastest{
		lock:        make(chan bool, 1),
		metricsChan: make(chan Metric, fastestMetricsChanBufferSize),
		providers:   map[string]*avgMetrics{},
	}

	for _, provider := range providers {
		fastest.providers[provider] = nil
	}

	go fastest.consumeMetrics()

	return &fastest
}
