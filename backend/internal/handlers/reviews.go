package handlers

import (
	"net/http"

	"mongoscale/internal/models"
	"mongoscale/internal/utils"
)

func (h *Handlers) createReview(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireIdentity(r)
	if err != nil {
		return err
	}
	var request models.CreateReviewRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	review, err := h.reviews.Create(r.Context(), identity.UserID, request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusCreated, review)
	return nil
}

func (h *Handlers) listReviews(w http.ResponseWriter, r *http.Request) error {
	productID, err := utils.ParseObjectID(r.PathValue("productId"))
	if err != nil {
		return err
	}
	limit, err := utils.IntQuery(r, "limit", 10, 1, 100)
	if err != nil {
		return err
	}
	page, err := utils.IntQuery(r, "page", 1, 1, 100000)
	if err != nil {
		return err
	}
	result, err := h.reviews.ListByProduct(r.Context(), productID, limit, page, r.URL.Query().Get("cursor"))
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, result)
	return nil
}

func (h *Handlers) reviewSummary(w http.ResponseWriter, r *http.Request) error {
	productID, err := utils.ParseObjectID(r.PathValue("productId"))
	if err != nil {
		return err
	}
	summary, err := h.reviews.Summary(r.Context(), productID)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, summary)
	return nil
}
