package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/NicoMartinns/gastano-menos/internal/service"
	"github.com/go-chi/chi/v5"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

type CreateTransactionRequest struct {
	CategoryID      string  `json:"category_id"`
	Description     string  `json:"description"`
	Amount          float64 `json:"amount"`
	Type            string  `json:"type"`
	Date            string  `json:"date"`
	IsRecurring     bool    `json:"is_recurring"`
	RecurringMonths *int    `json:"recurring_months"`
	RecurrenceDay   *int    `json:"recurrence_day"`
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "data inválida, use o formato YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	transaction := &domain.Transaction{
		UserID:          userID,
		CategoryID:      req.CategoryID,
		Description:     req.Description,
		Amount:          req.Amount,
		Type:            domain.TransactionType(req.Type),
		Date:            date,
		IsRecurring:     req.IsRecurring,
		RecurringMonths: req.RecurringMonths,
		RecurrenceDay:   req.RecurrenceDay,
	}

	if err := h.service.Create(transaction); err != nil {
		log.Printf("erro ao criar transação: %v", err)
		http.Error(w, "erro ao criar transação", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

func (h *TransactionHandler) GetByMonth(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "ano inválido", http.StatusBadRequest)
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		http.Error(w, "mês inválido", http.StatusBadRequest)
		return
	}

	transactions, err := h.service.GetByMonth(userID, year, month)
	if err != nil {
		log.Printf("erro ao buscar transações: %v", err)
		http.Error(w, "erro ao buscar transações", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "single"
	}

	if err := h.service.DeleteWithScope(id, userID, scope); err != nil {
		log.Printf("erro ao deletar transação: %v", err)
		http.Error(w, "erro ao deletar transação", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "single"
	}

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "data inválida", http.StatusBadRequest)
		return
	}

	transaction := &domain.Transaction{
		ID:          id,
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Description: req.Description,
		Amount:      req.Amount,
		Type:        domain.TransactionType(req.Type),
		Date:        date,
	}

	if err := h.service.UpdateWithScope(id, userID, scope, transaction); err != nil {
		log.Printf("erro ao atualizar transação: %v", err)
		http.Error(w, "erro ao atualizar transação", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

func (h *TransactionHandler) GetMonthlySummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(w, "usuário não autenticado", http.StatusUnauthorized)
		return
	}

	yearStr := r.URL.Query().Get("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "ano inválido", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetMonthlySummary(userID, year)
	if err != nil {
		log.Printf("erro ao buscar resumo mensal: %v", err)
		http.Error(w, "erro ao buscar resumo mensal", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}