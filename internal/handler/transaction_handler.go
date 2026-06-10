package handler

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
    "strconv"

    "github.com/NicoMartinns/gastano-menos/internal/domain"
    "github.com/NicoMartinns/gastano-menos/internal/service"
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