package limiter

import (
	"net/http"
	"sync"
	"time"

	"github.com/itstheanurag/executioner/internal/metrics"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	globalLimiter *rate.Limiter
	perIPLimiters sync.Map
	ipRate        rate.Limit
	ipBurst       int
	maxConcurrent int64
	currentConc   int64
	mu            sync.Mutex
}

func NewRateLimiter(globalRPS float64, perIPRPS float64, perIPBurst int, maxConcurrent int) *RateLimiter {
	return &RateLimiter{
		globalLimiter: rate.NewLimiter(rate.Limit(globalRPS), int(globalRPS)*2),
		ipRate:        rate.Limit(perIPRPS),
		ipBurst:       perIPBurst,
		maxConcurrent: int64(maxConcurrent),
	}
}

func (rl *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	if limiter, ok := rl.perIPLimiters.Load(ip); ok {
		return limiter.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(rl.ipRate, rl.ipBurst)
	rl.perIPLimiters.Store(ip, limiter)
	return limiter
}

func (rl *RateLimiter) Allow(ip string) bool {
	// Check global limit
	if !rl.globalLimiter.Allow() {
		metrics.RateLimitHits.Inc()
		return false
	}

	// Check per-IP limit
	ipLimiter := rl.getIPLimiter(ip)
	if !ipLimiter.Allow() {
		metrics.RateLimitHits.Inc()
		return false
	}

	// Check concurrent execution limit
	rl.mu.Lock()
	if rl.currentConc >= rl.maxConcurrent {
		rl.mu.Unlock()
		metrics.RateLimitHits.Inc()
		return false
	}
	rl.currentConc++
	rl.mu.Unlock()

	return true
}

func (rl *RateLimiter) Done() {
	rl.mu.Lock()
	if rl.currentConc > 0 {
		rl.currentConc--
	}
	rl.mu.Unlock()
}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		if !rl.Allow(ip) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		defer rl.Done()

		next(w, r)
	}
}

// CleanupOldLimiters removes IP limiters that haven't been used recently
func (rl *RateLimiter) StartCleanup(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			rl.perIPLimiters.Range(func(key, value any) bool {
				// Simple cleanup: remove all entries periodically
				// A more sophisticated approach would track last access time
				rl.perIPLimiters.Delete(key)
				return true
			})
		}
	}()
}
