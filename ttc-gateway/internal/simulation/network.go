package simulation

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// NetworkSimulator simulates realistic network conditions for space communications
type NetworkSimulator struct {
	mu                sync.RWMutex
	enabled           bool
	latencyMin        time.Duration
	latencyMax        time.Duration
	packetLossRate    float64 // 0.0 to 1.0
	jitterRange       time.Duration
	bandwidthLimitKBs int // KB/s
	stats             NetworkStats
}

// NetworkStats tracks network simulation statistics
type NetworkStats struct {
	TotalPackets     int64
	DroppedPackets   int64
	AverageLatencyMs float64
	MaxLatencyMs     float64
	BytesTransferred int64
}

// NetworkCondition represents different network condition presets
type NetworkCondition string

const (
	// LEO - Low Earth Orbit (typical: 550-2000 km altitude)
	LEO NetworkCondition = "leo"
	// MEO - Medium Earth Orbit (typical: 2000-35786 km altitude)
	MEO NetworkCondition = "meo"
	// GEO - Geostationary Orbit (35786 km altitude)
	GEO NetworkCondition = "geo"
	// DeepSpace - Beyond Earth orbit
	DeepSpace NetworkCondition = "deep_space"
	// Degraded - Simulates adverse conditions
	Degraded NetworkCondition = "degraded"
)

// NewNetworkSimulator creates a new network simulator
func NewNetworkSimulator() *NetworkSimulator {
	return &NetworkSimulator{
		enabled:           false,
		latencyMin:        10 * time.Millisecond,
		latencyMax:        50 * time.Millisecond,
		packetLossRate:    0.01, // 1%
		jitterRange:       5 * time.Millisecond,
		bandwidthLimitKBs: 1024, // 1 MB/s
	}
}

// SetCondition sets the network condition to a preset
func (ns *NetworkSimulator) SetCondition(condition NetworkCondition) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	switch condition {
	case LEO:
		// LEO: 20-40ms latency, 0.5% packet loss
		ns.latencyMin = 20 * time.Millisecond
		ns.latencyMax = 40 * time.Millisecond
		ns.packetLossRate = 0.005
		ns.jitterRange = 5 * time.Millisecond
		ns.bandwidthLimitKBs = 10240 // 10 MB/s

	case MEO:
		// MEO: 50-100ms latency, 1% packet loss
		ns.latencyMin = 50 * time.Millisecond
		ns.latencyMax = 100 * time.Millisecond
		ns.packetLossRate = 0.01
		ns.jitterRange = 10 * time.Millisecond
		ns.bandwidthLimitKBs = 5120 // 5 MB/s

	case GEO:
		// GEO: 240-280ms latency (round-trip ~500ms), 2% packet loss
		ns.latencyMin = 240 * time.Millisecond
		ns.latencyMax = 280 * time.Millisecond
		ns.packetLossRate = 0.02
		ns.jitterRange = 20 * time.Millisecond
		ns.bandwidthLimitKBs = 2048 // 2 MB/s

	case DeepSpace:
		// Deep Space: seconds to minutes of latency
		ns.latencyMin = 2 * time.Second
		ns.latencyMax = 5 * time.Second
		ns.packetLossRate = 0.05
		ns.jitterRange = 500 * time.Millisecond
		ns.bandwidthLimitKBs = 128 // 128 KB/s

	case Degraded:
		// Degraded: High latency, high packet loss (e.g., during solar storm)
		ns.latencyMin = 100 * time.Millisecond
		ns.latencyMax = 500 * time.Millisecond
		ns.packetLossRate = 0.15 // 15%
		ns.jitterRange = 100 * time.Millisecond
		ns.bandwidthLimitKBs = 256 // 256 KB/s
	}
}

// Enable enables network simulation
func (ns *NetworkSimulator) Enable() {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.enabled = true
}

// Disable disables network simulation
func (ns *NetworkSimulator) Disable() {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.enabled = false
}

// SimulatePacket simulates sending a packet through the network
// Returns (success, latency, error)
func (ns *NetworkSimulator) SimulatePacket(sizeBytes int) (bool, time.Duration, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.enabled {
		return true, 0, nil
	}

	ns.stats.TotalPackets++
	ns.stats.BytesTransferred += int64(sizeBytes)

	// Simulate packet loss
	if rand.Float64() < ns.packetLossRate {
		ns.stats.DroppedPackets++
		return false, 0, fmt.Errorf("packet dropped (simulated loss)")
	}

	// Calculate latency with jitter
	baseLatency := ns.latencyMin + time.Duration(rand.Int63n(int64(ns.latencyMax-ns.latencyMin)))
	jitter := time.Duration(rand.Int63n(int64(ns.jitterRange))) - ns.jitterRange/2
	latency := baseLatency + jitter

	// Update stats
	latencyMs := float64(latency.Milliseconds())
	if latencyMs > ns.stats.MaxLatencyMs {
		ns.stats.MaxLatencyMs = latencyMs
	}
	// Running average
	totalPackets := float64(ns.stats.TotalPackets - ns.stats.DroppedPackets)
	ns.stats.AverageLatencyMs = (ns.stats.AverageLatencyMs*(totalPackets-1) + latencyMs) / totalPackets

	// Simulate bandwidth limit (simplified)
	transmissionTime := time.Duration(sizeBytes/ns.bandwidthLimitKBs) * time.Millisecond
	totalDelay := latency + transmissionTime

	return true, totalDelay, nil
}

// SimulateDelay simulates network delay (blocking)
func (ns *NetworkSimulator) SimulateDelay(sizeBytes int) error {
	success, delay, err := ns.SimulatePacket(sizeBytes)
	if !success {
		return err
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	return nil
}

// GetStats returns current network statistics
func (ns *NetworkSimulator) GetStats() NetworkStats {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return ns.stats
}

// ResetStats resets network statistics
func (ns *NetworkSimulator) ResetStats() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.stats = NetworkStats{}
}

// IsEnabled returns whether network simulation is enabled
func (ns *NetworkSimulator) IsEnabled() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return ns.enabled
}

// GetPacketLossRate returns the current packet loss rate
func (ns *NetworkSimulator) GetPacketLossRate() float64 {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if ns.stats.TotalPackets == 0 {
		return 0
	}

	return float64(ns.stats.DroppedPackets) / float64(ns.stats.TotalPackets)
}

