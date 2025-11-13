package profiling

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type MemoryProfiler struct {
	logger           *logrus.Logger
	baselineStats    *runtime.MemStats
	mu               sync.RWMutex
	enabled          bool
	samplingInterval time.Duration
	alerts           []MemoryAlert
}

type MemoryStats struct {
	Timestamp    time.Time `json:"timestamp"`
	AllocMB      float64   `json:"alloc_mb"`       // Currently allocated memory
	TotalAllocMB float64   `json:"total_alloc_mb"` // Total allocated over time
	SysMB        float64   `json:"sys_mb"`         // Total memory obtained from OS
	NumGC        uint32    `json:"num_gc"`         // Number of GC cycles
	HeapObjects  uint64    `json:"heap_objects"`   // Number of allocated heap objects
	GoroutineNum int       `json:"goroutine_num"`  // Number of goroutines
	GCPauseMs    float64   `json:"gc_pause_ms"`    // Last GC pause duration in ms
}

type MemoryAlert struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "leak", "high_usage", "gc_pressure"
	Description string    `json:"description"`
	AllocMB     float64   `json:"alloc_mb"`
	ThresholdMB float64   `json:"threshold_mb"`
}

type MemorySnapshot struct {
	Stats      *MemoryStats   `json:"stats"`
	Alerts     []MemoryAlert  `json:"alerts"`
	LeakStatus *LeakDetection `json:"leak_status,omitempty"`
}

type LeakDetection struct {
	HasLeak         bool    `json:"has_leak"`
	GrowthRateMBMin float64 `json:"growth_rate_mb_min"`
	CheckDuration   string  `json:"check_duration"`
	Recommendation  string  `json:"recommendation"`
}

func NewMemoryProfiler(logger *logrus.Logger) *MemoryProfiler {
	profiler := &MemoryProfiler{
		logger:           logger,
		enabled:          true,
		samplingInterval: 30 * time.Second,
		alerts:           make([]MemoryAlert, 0),
	}

	// Capture baseline stats
	profiler.baselineStats = &runtime.MemStats{}
	runtime.ReadMemStats(profiler.baselineStats)

	return profiler
}

// Start begins periodic memory monitoring
func (p *MemoryProfiler) Start() {
	go p.monitorMemory()
}

// GetMemoryStats returns current memory statistics
func (p *MemoryProfiler) GetMemoryStats() *MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	gcPauseMs := float64(0)
	if m.NumGC > 0 && len(m.PauseNs) > 0 {
		// Get the most recent GC pause
		gcPauseMs = float64(m.PauseNs[(m.NumGC+255)%256]) / 1e6
	}

	return &MemoryStats{
		Timestamp:    time.Now(),
		AllocMB:      float64(m.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(m.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(m.Sys) / 1024 / 1024,
		NumGC:        m.NumGC,
		HeapObjects:  m.HeapObjects,
		GoroutineNum: runtime.NumGoroutine(),
		GCPauseMs:    gcPauseMs,
	}
}

// DetectLeaks analyzes memory patterns for potential leaks
func (p *MemoryProfiler) DetectLeaks(duration time.Duration) *LeakDetection {
	initialStats := p.GetMemoryStats()

	// Wait for specified duration
	time.Sleep(duration)

	// Force GC to get accurate readings
	runtime.GC()
	debug.FreeOSMemory()
	time.Sleep(100 * time.Millisecond)

	finalStats := p.GetMemoryStats()

	// Calculate growth rate
	memoryGrowth := finalStats.AllocMB - initialStats.AllocMB
	growthRate := memoryGrowth / duration.Minutes()

	detection := &LeakDetection{
		HasLeak:         false,
		GrowthRateMBMin: growthRate,
		CheckDuration:   duration.String(),
	}

	// Thresholds for leak detection
	const (
		suspiciousGrowthRate = 1.0  // 1 MB/min
		criticalGrowthRate   = 5.0  // 5 MB/min
		objectGrowthRate     = 1000 // 1000 objects/min
	)

	objectGrowth := float64(finalStats.HeapObjects-initialStats.HeapObjects) / duration.Minutes()

	if growthRate > criticalGrowthRate {
		detection.HasLeak = true
		detection.Recommendation = fmt.Sprintf(
			"Critical memory leak detected! Memory growing at %.2f MB/min. Investigate immediately.",
			growthRate,
		)
	} else if growthRate > suspiciousGrowthRate || objectGrowth > objectGrowthRate {
		detection.HasLeak = true
		detection.Recommendation = fmt.Sprintf(
			"Potential memory leak. Memory: %.2f MB/min, Objects: %.0f/min. Monitor closely.",
			growthRate, objectGrowth,
		)
	} else {
		detection.Recommendation = "Memory usage appears stable. No leak detected."
	}

	// Log findings
	if detection.HasLeak {
		p.logger.WithFields(logrus.Fields{
			"growth_rate_mb_min": growthRate,
			"object_growth_rate": objectGrowth,
			"duration":           duration.String(),
		}).Warn(detection.Recommendation)

		// Add alert
		p.addAlert(MemoryAlert{
			Timestamp:   time.Now(),
			Type:        "leak",
			Description: detection.Recommendation,
			AllocMB:     finalStats.AllocMB,
		})
	}

	return detection
}

// monitorMemory runs periodic memory checks
func (p *MemoryProfiler) monitorMemory() {
	ticker := time.NewTicker(p.samplingInterval)
	defer ticker.Stop()

	const (
		highMemoryThreshold = 100.0 // MB
		gcPressureThreshold = 10    // GCs per minute
	)

	lastGCCount := uint32(0)
	lastCheckTime := time.Now()

	for range ticker.C {
		if !p.enabled {
			continue
		}

		stats := p.GetMemoryStats()

		// Check for high memory usage
		if stats.AllocMB > highMemoryThreshold {
			p.addAlert(MemoryAlert{
				Timestamp:   stats.Timestamp,
				Type:        "high_usage",
				Description: fmt.Sprintf("High memory usage: %.2f MB allocated", stats.AllocMB),
				AllocMB:     stats.AllocMB,
				ThresholdMB: highMemoryThreshold,
			})
		}

		// Check for GC pressure
		timeDelta := time.Since(lastCheckTime).Minutes()
		gcRate := float64(stats.NumGC-lastGCCount) / timeDelta

		if gcRate > gcPressureThreshold {
			p.addAlert(MemoryAlert{
				Timestamp:   stats.Timestamp,
				Type:        "gc_pressure",
				Description: fmt.Sprintf("High GC rate: %.2f GCs/min", gcRate),
				AllocMB:     stats.AllocMB,
			})
		}

		lastGCCount = stats.NumGC
		lastCheckTime = stats.Timestamp

		// Log stats periodically
		p.logger.WithFields(logrus.Fields{
			"alloc_mb":     stats.AllocMB,
			"sys_mb":       stats.SysMB,
			"goroutines":   stats.GoroutineNum,
			"heap_objects": stats.HeapObjects,
			"gc_pause_ms":  stats.GCPauseMs,
		}).Debug("Memory statistics")
	}
}

// addAlert adds an alert to the history
func (p *MemoryProfiler) addAlert(alert MemoryAlert) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.alerts = append(p.alerts, alert)

	// Keep only last 100 alerts
	if len(p.alerts) > 100 {
		p.alerts = p.alerts[len(p.alerts)-100:]
	}
}

// GetAlerts returns recent memory alerts
func (p *MemoryProfiler) GetAlerts(since time.Time) []MemoryAlert {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var filtered []MemoryAlert
	for _, alert := range p.alerts {
		if alert.Timestamp.After(since) {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// GetSnapshot returns a complete memory snapshot
func (p *MemoryProfiler) GetSnapshot() *MemorySnapshot {
	stats := p.GetMemoryStats()
	alerts := p.GetAlerts(time.Now().Add(-1 * time.Hour))

	// Quick leak check (1 minute)
	leakStatus := p.DetectLeaks(1 * time.Minute)

	return &MemorySnapshot{
		Stats:      stats,
		Alerts:     alerts,
		LeakStatus: leakStatus,
	}
}

// HTTP Handlers

// MemoryHandler serves current memory statistics
func (p *MemoryProfiler) MemoryHandler(w http.ResponseWriter, r *http.Request) {
	stats := p.GetMemoryStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// SnapshotHandler serves complete memory snapshot
func (p *MemoryProfiler) SnapshotHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := p.GetSnapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// HeapHandler serves heap profile
func (p *MemoryProfiler) HeapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="heap.pprof"`)
	if err := pprof.WriteHeapProfile(w); err != nil {
		p.logger.WithError(err).Error("Failed to write heap profile")
		http.Error(w, "Failed to generate heap profile", http.StatusInternalServerError)
	}
}

// GoroutineHandler serves goroutine profile
func (p *MemoryProfiler) GoroutineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if err := pprof.Lookup("goroutine").WriteTo(w, 2); err != nil {
		p.logger.WithError(err).Error("Failed to write goroutine profile")
		http.Error(w, "Failed to generate goroutine profile", http.StatusInternalServerError)
	}
}

// ForceGC forces garbage collection
func (p *MemoryProfiler) ForceGC() *MemoryStats {
	beforeStats := p.GetMemoryStats()

	runtime.GC()
	debug.FreeOSMemory()

	afterStats := p.GetMemoryStats()

	p.logger.WithFields(logrus.Fields{
		"before_mb": beforeStats.AllocMB,
		"after_mb":  afterStats.AllocMB,
		"freed_mb":  beforeStats.AllocMB - afterStats.AllocMB,
	}).Info("Forced garbage collection")

	return afterStats
}

// SetEnabled enables or disables monitoring
func (p *MemoryProfiler) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}
