package handlers

import (
	"net/http"

	"mongoscale/internal/models"
	"mongoscale/internal/services"
	"mongoscale/internal/utils"
)

func (h *Handlers) createOrder(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	var request models.CreateOrderRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	order, err := h.orders.Create(r.Context(), identity.UserID, request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusCreated, order)
	return nil
}

func (h *Handlers) listOrders(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	limit, err := utils.IntQuery(r, "limit", 20, 1, 100)
	if err != nil {
		return err
	}
	page, err := utils.IntQuery(r, "page", 1, 1, 100000)
	if err != nil {
		return err
	}
	query := services.ListOrdersQuery{
		Status: r.URL.Query().Get("status"),
		UserID: r.URL.Query().Get("userId"),
		Limit:  limit,
		Page:   page,
		Cursor: r.URL.Query().Get("cursor"),
	}
	result, err := h.orders.List(r.Context(), identity, query)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, result)
	return nil
}

func (h *Handlers) orderSummary(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	rows, err := h.orders.Summary(r.Context(), identity)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, rows)
	return nil
}

func (h *Handlers) getOrder(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	id, err := utils.ParseObjectID(r.PathValue("id"))
	if err != nil {
		return err
	}
	order, err := h.orders.Get(r.Context(), identity, id)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, order)
	return nil
}

func (h *Handlers) updateOrderStatus(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	id, err := utils.ParseObjectID(r.PathValue("id"))
	if err != nil {
		return err
	}
	var request models.UpdateOrderStatusRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	order, err := h.orders.UpdateStatus(r.Context(), identity, id, request.Status)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, order)
	return nil
}
