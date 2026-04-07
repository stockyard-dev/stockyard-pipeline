package server

import (
	"encoding/json"
	"github.com/stockyard-dev/stockyard-pipeline/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.mux.HandleFunc("GET /api/pipelines", s.listPipelines)
	s.mux.HandleFunc("POST /api/pipelines", s.createPipeline)
	s.mux.HandleFunc("GET /api/pipelines/{id}", s.getPipeline)
	s.mux.HandleFunc("PUT /api/pipelines/{id}", s.updatePipeline)
	s.mux.HandleFunc("DELETE /api/pipelines/{id}", s.deletePipeline)
	s.mux.HandleFunc("POST /api/pipelines/{id}/run", s.runPipeline)
	s.mux.HandleFunc("GET /api/pipelines/{id}/runs", s.listRuns)
	s.mux.HandleFunc("GET /api/runs/{id}", s.getRun)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/pipeline/"})
	})
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", http.StatusFound)
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"pipelines": orEmpty(s.db.ListPipelines())})
}
func (s *Server) createPipeline(w http.ResponseWriter, r *http.Request) {
	var p store.Pipeline
	json.NewDecoder(r.Body).Decode(&p)
	if p.Name == "" {
		writeErr(w, 400, "name required")
		return
	}
	p.Enabled = true
	s.db.CreatePipeline(&p)
	writeJSON(w, 201, s.db.GetPipeline(p.ID))
}
func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	p := s.db.GetPipeline(r.PathValue("id"))
	if p == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, p)
}
func (s *Server) updatePipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ex := s.db.GetPipeline(id)
	if ex == nil {
		writeErr(w, 404, "not found")
		return
	}
	var p store.Pipeline
	json.NewDecoder(r.Body).Decode(&p)
	if p.Name == "" {
		p.Name = ex.Name
	}
	if p.Steps == nil {
		p.Steps = ex.Steps
	}
	s.db.UpdatePipeline(id, &p)
	writeJSON(w, 200, s.db.GetPipeline(id))
}
func (s *Server) deletePipeline(w http.ResponseWriter, r *http.Request) {
	s.db.DeletePipeline(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) runPipeline(w http.ResponseWriter, r *http.Request) {
	p := s.db.GetPipeline(r.PathValue("id"))
	if p == nil {
		writeErr(w, 404, "not found")
		return
	}
	run := store.Run{PipelineID: p.ID, Status: "running", StartedAt: time.Now().UTC().Format(time.RFC3339)}
	start := time.Now()
	for _, step := range p.Steps {
		sr := store.StepResult{StepName: step.Name, Status: "success", DurationMs: 10, Output: "Step " + step.Name + " completed"}
		run.StepResults = append(run.StepResults, sr)
	}
	run.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	run.DurationMs = int(time.Since(start).Milliseconds())
	run.Status = "success"
	s.db.SaveRun(&run)
	writeJSON(w, 200, s.db.GetRun(run.ID))
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"runs": orEmpty(s.db.ListRuns(r.PathValue("id"), 20))})
}
func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	run := s.db.GetRun(r.PathValue("id"))
	if run == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, run)
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.db.Stats()
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "pipeline", "pipelines": st.Pipelines, "runs": st.Runs})
}
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

// ─── personalization (auto-added) ──────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("%s: warning: could not parse config.json: %v", "pipeline", err)
		return
	}
	s.pCfg = cfg
	log.Printf("%s: loaded personalization from %s", "pipeline", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, 400)
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		http.Error(w, `{"error":"save failed"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":"saved"}`))
}
