package handlers

import (
	"net/http"

	"mongoscale/internal/models"
	"mongoscale/internal/utils"
)

func (h *Handlers) register(w http.ResponseWriter, r *http.Request) error {
	var request models.RegisterRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	response, err := h.auth.Register(r.Context(), request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) error {
	var request models.LoginRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	response, err := h.auth.Login(r.Context(), request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *Handlers) me(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	user, err := h.auth.Me(r.Context(), identity.UserID)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, user)
	return nil
}
