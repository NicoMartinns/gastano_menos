package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/NicoMartinns/gastano-menos/internal/service"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct{ service *service.UserService }

func NewUserHandler(s *service.UserService) *UserHandler { return &UserHandler{service: s} }

type UserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
}

func writeUserError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidUser):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrUserNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrEmailInUse), errors.Is(err, domain.ErrLastAdmin), errors.Is(err, domain.ErrSelfAccessChange):
		status = http.StatusConflict
	}
	if status == http.StatusInternalServerError {
		log.Printf("erro na gestão de usuários: %v", err)
		http.Error(w, "erro ao processar usuário", status)
		return
	}
	http.Error(w, err.Error(), status)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := 20, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil || limit < 1 || limit > 100 {
		http.Error(w, "limit deve estar entre 1 e 100", http.StatusBadRequest)
		return
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
	}
	if err != nil || offset < 0 {
		http.Error(w, "offset inválido", http.StatusBadRequest)
		return
	}
	users, total, err := h.service.List(limit, offset)
	if err != nil {
		writeUserError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users, "total": total})
}

func decodeUser(w http.ResponseWriter, r *http.Request) (*UserRequest, bool) {
	var req UserRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || req.IsActive == nil {
		http.Error(w, "payload inválido; informe name, email, role e is_active", http.StatusBadRequest)
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "payload deve conter um único objeto JSON", http.StatusBadRequest)
		return nil, false
	}
	return &req, true
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeUser(w, r)
	if !ok {
		return
	}
	user := &domain.User{Name: req.Name, Email: req.Email, Role: req.Role, IsActive: *req.IsActive}
	if err := h.service.Create(user, req.Password); err != nil {
		writeUserError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.ToLower(chi.URLParam(r, "id"))
	if !uuidPattern.MatchString(id) {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	req, ok := decodeUser(w, r)
	if !ok {
		return
	}
	if req.Password != "" {
		http.Error(w, "alteração de senha não suportada nesta operação", http.StatusBadRequest)
		return
	}
	actor, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}
	user := &domain.User{ID: id, Name: req.Name, Email: req.Email, Role: req.Role, IsActive: *req.IsActive}
	if err := h.service.Update(actor, user); err != nil {
		writeUserError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
