package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/NicoMartinns/gastano-menos/internal/service"
	"github.com/go-chi/chi/v5"
)

type CategoryHandler struct {
	service *service.CategoryService
}

func NewCategoryHandler(service *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	categories, err := h.service.GetAll(userID)
	if err != nil {
		log.Printf("erro ao buscar categorias: %v", err)
		http.Error(w, "erro ao buscar categorias", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

type CreateCategoryRequest struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	ParentID *string `json:"parent_id"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	category, err := h.service.Create(userID, req.Name, req.Type, req.ParentID)
	if err != nil {
		log.Printf("erro ao criar categoria: %v", err)
		http.Error(w, "erro ao criar categoria", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")

	if err := h.service.Delete(id, userID); err != nil {
		log.Printf("erro ao deletar categoria: %v", err)
		http.Error(w, "erro ao deletar categoria", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}