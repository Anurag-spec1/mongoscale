package handlers

import (
	"net/http"

	"mongoscale/internal/models"
	"mongoscale/internal/repository"
	"mongoscale/internal/utils"
)

var allowedSorts = map[string]bool{"price_asc": true, "price_desc": true, "rating": true, "newest": true}

func productFilter(r *http.Request, pathCategory string) (repository.ProductFilter, error) {
	var filter repository.ProductFilter
	limit, err := utils.IntQuery(r, "limit", 20, 1, 100)
	if err != nil {
		return filter, err
	}
	page, err := utils.IntQuery(r, "page", 1, 1, 100000)
	if err != nil {
		return filter, err
	}
	query, err := utils.TrimmedQuery(r, "q", 100)
	if err != nil {
		return filter, err
	}
	category, err := utils.TrimmedQuery(r, "category", 60)
	if err != nil {
		return filter, err
	}
	if pathCategory != "" {
		category = pathCategory
	}
	minPrice, err := utils.FloatQuery(r, "minPrice")
	if err != nil {
		return filter, err
	}
	maxPrice, err := utils.FloatQuery(r, "maxPrice")
	if err != nil {
		return filter, err
	}
	sortKey := r.URL.Query().Get("sort")
	if sortKey == "" {
		sortKey = "price_asc"
	}
	if !allowedSorts[sortKey] {
		return filter, utils.BadRequest("sort must be one of price_asc, price_desc, rating, newest")
	}
	return repository.ProductFilter{
		Category: category,
		Query:    query,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		Sort:     sortKey,
		Limit:    limit,
		Page:     page,
		Cursor:   r.URL.Query().Get("cursor"),
	}, nil
}

func (h *Handlers) listProducts(w http.ResponseWriter, r *http.Request) error {
	filter, err := productFilter(r, "")
	if err != nil {
		return err
	}
	page, err := h.products.List(r.Context(), filter)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, page)
	return nil
}

func (h *Handlers) listProductsByCategory(w http.ResponseWriter, r *http.Request) error {
	category := r.PathValue("category")
	if category == "" {
		return utils.BadRequest("category is required")
	}
	filter, err := productFilter(r, category)
	if err != nil {
		return err
	}
	page, err := h.products.List(r.Context(), filter)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, page)
	return nil
}

func (h *Handlers) listCategories(w http.ResponseWriter, r *http.Request) error {
	categories, err := h.products.Categories(r.Context())
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, categories)
	return nil
}

func (h *Handlers) getProduct(w http.ResponseWriter, r *http.Request) error {
	id, err := utils.ParseObjectID(r.PathValue("id"))
	if err != nil {
		return err
	}
	product, err := h.products.Get(r.Context(), id)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, product)
	return nil
}

func (h *Handlers) createProduct(w http.ResponseWriter, r *http.Request) error {
	var request models.CreateProductRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	product, err := h.products.Create(r.Context(), request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusCreated, product)
	return nil
}

func (h *Handlers) adjustStock(w http.ResponseWriter, r *http.Request) error {
	id, err := utils.ParseObjectID(r.PathValue("id"))
	if err != nil {
		return err
	}
	var request models.StockAdjustmentRequest
	if err := decodeBody(w, r, &request); err != nil {
		return err
	}
	product, err := h.products.AdjustStock(r.Context(), id, request)
	if err != nil {
		return err
	}
	utils.WriteJSON(w, http.StatusOK, product)
	return nil
}
