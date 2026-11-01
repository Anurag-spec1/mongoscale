package handlers

import (
	"errors"
	"net/http"

	"mongoscale/internal/database"
	"mongoscale/internal/models"
	"mongoscale/internal/utils"
)

type optimizationCatalog struct {
	Current models.Mode     `json:"current"`
	Tuning  models.Tuning   `json:"tuning"`
	Modes   []models.Tuning `json:"modes"`
}

func (h *Handlers) health(w http.ResponseWriter, r *http.Request) error {
	report, healthy := h.system.Health(r.Context())
	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	utils.WriteJSON(w, status, report)
	return nil
}

func (h *Handlers) metrics(w http.ResponseWriter, r *http.Request) error {
	history, err := utils.IntQuery(r, "history", 120, 1, 600)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, h.system.Metrics(history))
	return nil
}

func (h *Handlers) prometheus(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	h.system.WritePrometheus(w)
	return nil
}

func (h *Handlers) mongoStatus(w http.ResponseWriter, r *http.Request) error {
	utils.WriteJSON(w, http.StatusOK, h.system.MongoStatus(r.Context()))
	return nil
}

func (h *Handlers) optimizations(w http.ResponseWriter, r *http.Request) error {
	tuning := h.optimization.Tuning()
	utils.WriteJSON(w, http.StatusOK, optimizationCatalog{
		Current: tuning.Mode,
		Tuning:  tuning,
		Modes:   models.AllTunings(),
	})
	return nil
}

func (h *Handlers) toggleOptimization(w http.ResponseWriter, r *http.Request) error {
	mode, err := models.ParseMode(r.PathValue("feature"))
	if err != nil {
		return utils.BadRequest("feature must be one of unoptimized, indexed, replica-set, sharded, fully-optimized")
	}
	change, err := h.optimization.Apply(r.Context(), mode)
	if err != nil {
		return err
	}
	h.system.InvalidateStatus()
	utils.WriteJSON(w, http.StatusOK, change)
	return nil
}

func (h *Handlers) startLoadTest(w http.ResponseWriter, r *http.Request) error {
	var request models.LoadTestRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	status, err := h.loadTest.Start(request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusAccepted, status)
	return nil
}

func (h *Handlers) stopLoadTest(w http.ResponseWriter, r *http.Request) error {
	h.loadTest.Stop()
	utils.WriteJSON(w, http.StatusAccepted, h.loadTest.Status())
	return nil
}

func (h *Handlers) loadTestStatus(w http.ResponseWriter, r *http.Request) error {
	utils.WriteJSON(w, http.StatusOK, h.loadTest.Status())
	return nil
}

func (h *Handlers) loadTestResults(w http.ResponseWriter, r *http.Request) error {
	results, err := h.loadTest.Results(r.Context())
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, results)
	return nil
}

func (h *Handlers) stepDown(w http.ResponseWriter, r *http.Request) error {
	err := h.system.StepDown(r.Context(), r.URL.Query().Get("shard"))
	if errors.Is(err, database.ErrUnsupportedTopology) {
		return utils.Conflict("failover demo needs the replica-set or sharded compose file")
	}
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusAccepted, map[string]string{"message": "primary stepped down, an election is in progress"})
	return nil
}

func (h *Handlers) notFound(w http.ResponseWriter, r *http.Request) error {
	return utils.NotFound("route not found")
}
