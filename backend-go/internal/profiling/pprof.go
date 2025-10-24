package profiling

import (
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
)

// RegisterPProfHandlers registers pprof profiling endpoints
// These endpoints should be protected with authentication in production
func RegisterPProfHandlers(r chi.Router) {
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Additional handlers for heap, goroutine, threadcreate, block
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))
	r.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	r.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
}

// PProfConfig holds pprof configuration
type PProfConfig struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

// ProfileHandler returns a middleware that enables profiling
func ProfileHandler(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				http.Error(w, "Profiling is disabled", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

/*
Usage Documentation:

1. CPU Profiling:
   go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

2. Heap Profiling:
   go tool pprof http://localhost:8080/debug/pprof/heap

3. Goroutine Profiling:
   go tool pprof http://localhost:8080/debug/pprof/goroutine

4. Block Profiling:
   go tool pprof http://localhost:8080/debug/pprof/block

5. Mutex Profiling:
   go tool pprof http://localhost:8080/debug/pprof/mutex

6. Download and analyze:
   curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
   go tool pprof cpu.prof

7. Web interface:
   go tool pprof -http=:8081 cpu.prof

Common commands in pprof:
- top: Show top CPU/memory consumers
- list <function>: Show source code with annotations
- web: Generate call graph visualization
- pdf: Export call graph to PDF
- flamegraph: Generate flame graph (requires additional tools)
*/
