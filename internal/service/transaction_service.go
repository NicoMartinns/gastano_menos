package service

import (
    "github.com/NicoMartinns/gastano-menos/internal/domain"
    "github.com/NicoMartinns/gastano-menos/internal/repository"
)

type TransactionService struct {
    repo *repository.TransactionRepository
}

func NewTransactionService(repo *repository.TransactionRepository) *TransactionService {
    return &TransactionService{repo: repo}
}

func (s *TransactionService) Create(t *domain.Transaction) error {
    if err := s.repo.Create(t); err != nil {
        return err
    }

    if t.IsRecurring {
        if err := s.generateRecurring(t); err != nil {
            return err
        }
    }

    return nil
}

func (s *TransactionService) generateRecurring(origin *domain.Transaction) error {
    months := 24 // padrão para recorrência sem prazo (2 anos)
    if origin.RecurringMonths != nil {
        months = *origin.RecurringMonths
    }

    for i := 1; i < months; i++ {
        child := &domain.Transaction{
            UserID:            origin.UserID,
            CategoryID:        origin.CategoryID,
            Description:       origin.Description,
            Amount:            origin.Amount,
            Type:              origin.Type,
            Date:              origin.Date.AddDate(0, i, 0),
            IsRecurring:       false,
            RecurringOriginID: &origin.ID,
			PaymentMethod: origin.PaymentMethod,
        }

        if err := s.repo.Create(child); err != nil {
            return err
        }
    }

    return nil
}

func (s *TransactionService) GetByMonth(userID string, year int, month int) ([]domain.Transaction, error) {
	return s.repo.FindByMonth(userID, year, month)
}

func (s *TransactionService) Delete(id string, userID string) error {
	return s.repo.Delete(id, userID)
}

func (s *TransactionService) Update(t *domain.Transaction) error {
	return s.repo.Update(t)
}

func (s *TransactionService) GetMonthlySummary(userID string, year int) ([]domain.MonthlySummary, error) {
	return s.repo.GetMonthlySummary(userID, year)
}

func (s *TransactionService) DeleteWithScope(id string, userID string, scope string) error {
	t, err := s.repo.FindByID(id, userID)
	if err != nil {
		return err
	}

	if scope == "future" {
		originID := id
		if t.RecurringOriginID != nil {
			originID = *t.RecurringOriginID
		}
		return s.repo.DeleteFuture(id, userID, t.Date, originID)
	}

	return s.repo.Delete(id, userID)
}

func (s *TransactionService) UpdateWithScope(id string, userID string, scope string, t *domain.Transaction) error {
	existing, err := s.repo.FindByID(id, userID)
	if err != nil {
		return err
	}

	if scope == "future" {
		originID := id
		if existing.RecurringOriginID != nil {
			originID = *existing.RecurringOriginID
		}
		return s.repo.UpdateFuture(originID, userID, existing.Date, t)
	}

	return s.repo.Update(t)
}