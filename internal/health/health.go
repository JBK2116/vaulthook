// Package health provides a health check that covers the application and all it's dependencies.
package health

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	HEALTHY   = "healthy"
	UNHEALTHY = "unhealthy"
)

// HealthCheck holds the state of the most recent health check.
type HealthCheck struct {
	Status        string                     `json:"status"`
	UptimeSeconds int64                      `json:"uptime_seconds"`
	Version       string                     `json:"version"`
	Timestamp     string                     `json:"timestamp"`
	Checks        map[string]DependencyCheck `json:"checks"`
	Resources     ResourceStats              `json:"resources"`
}

// DependencyCheck is a single dependency check for a resource.
type DependencyCheck struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
}

// ResourceStats is a collection of resource usage statistics.
type ResourceStats struct {
	AllocMB      uint64 `json:"alloc_mb"`
	TotalAllocMB uint64 `json:"total_alloc_mb"`
	SysMB        uint64 `json:"sys_mb"`
	NumGoroutine uint64 `json:"num_goroutine"`
	NumGC        uint64 `json:"num_gc"`
	DiskFreeMB   uint64 `json:"disk_free_mb"`
}

var startTime time.Time

// HealthService is a health check service that provides a health check for the application and it's dependencies.
type HealthService struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

// InitStartTime tracks the start time of the server.
//
// It is to be called in app.go
func InitStartTime() {
	startTime = time.Now()
}

// NewHealthService returns a health service configured with the provided db & redis connection
func NewHealthService(pool *pgxpool.Pool, rdb *redis.Client) *HealthService {
	return &HealthService{
		db:  pool,
		rdb: rdb,
	}
}

// GetHealthCheck returns the current health check for the application.
func (h *HealthService) GetHealthCheck(ctx context.Context) HealthCheck {
	var health HealthCheck
	health.Checks = make(map[string]DependencyCheck)
	health.UptimeSeconds = getUpTime()
	health.Version = getVersion()
	health.Timestamp = getTimeStamp()

	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(2)

	go func() {
		defer wg.Done()
		check := h.checkDB(ctx)
		mu.Lock()
		health.Checks["postgres"] = check
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		check := h.checkRedis(ctx)
		mu.Lock()
		health.Checks["redis"] = check
		mu.Unlock()

	}()

	wg.Wait()

	health.Resources = getResources()
	if health.Checks["postgres"].Status != HEALTHY || health.Checks["redis"].Status != HEALTHY {
		health.Status = UNHEALTHY
	} else {
		health.Status = HEALTHY
	}
	return health
}

// checkRedis returns a health check for the redis dependency.
func (h *HealthService) checkRedis(ctx context.Context) DependencyCheck {
	var check DependencyCheck
	ctxR, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	start := time.Now()
	err := h.rdb.Ping(ctxR).Err()
	if err != nil {
		check.Status = UNHEALTHY
	} else {
		check.Status = HEALTHY
	}
	late := time.Since(start).Milliseconds()
	check.LatencyMS = late
	return check
}

// checkDB returns a health check for the postgresql dependency.
func (h *HealthService) checkDB(ctx context.Context) DependencyCheck {
	var check DependencyCheck
	ctxD, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	start := time.Now()
	err := h.db.Ping(ctxD)
	if err != nil {
		check.Status = UNHEALTHY
	} else {
		check.Status = HEALTHY
	}
	late := time.Since(start).Milliseconds()
	check.LatencyMS = late
	return check
}

// getUpTime returns the amount of seconds that the server has been running since the last reset.
func getUpTime() int64 {
	diff := time.Since(startTime).Seconds()
	return int64(diff)
}

// getTimeStamp returns the current time in ISO8601 format
func getTimeStamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// getVersion reads the vcs revision from the build info and returns it as a string.
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return "unknown"
}

// getResources returns the current resource usage statistics.
func getResources() ResourceStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var fs syscall.Statfs_t

	var diskFreeMB uint64
	if err := syscall.Statfs("/", &fs); err != nil {
		diskFreeMB = 0
	} else {
		diskFreeMB = (fs.Bavail * uint64(fs.Bsize)) / 1024 / 1024
	}

	res := ResourceStats{
		AllocMB:      m.Alloc / 1024 / 1024,
		TotalAllocMB: m.TotalAlloc / 1024 / 1024,
		SysMB:        m.Sys / 1024 / 1024,
		NumGoroutine: uint64(runtime.NumGoroutine()),
		NumGC:        uint64(m.NumGC),
		DiskFreeMB:   diskFreeMB,
	}
	return res
}
