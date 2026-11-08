package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Domain sentinel errors
var (
	ErrInvalidAmount    = errors.New("transaction amount must be strictly positive")
	ErrUnsupportedCurr  = errors.New("unsupported currency code")
	ErrTransientNetwork = errors.New("temporary network timeout communicating with ledger")
	ErrAccountSuspended = errors.New("account suspended due to compliance hold")
)

// Currency conversion rates against base USD
var fxRates = map[string]float64{
	"USD": 1.0,
	"EUR": 1.08,
	"GBP": 1.27,
	"JPY": 0.0065,
}

// TransactionEvent represents an incoming raw financial transaction payload.
type TransactionEvent struct {
	ID                  string
	AccountID           string
	Amount              float64
	Currency            string
	Timestamp           time.Time
	SimulateTransient   bool // For testing resilience & retry backoff
	SimulatePanic       bool // For testing worker panic isolation
}

// EnrichedTransaction represents the validated, normalized, and scored output record.
type EnrichedTransaction struct {
	TransactionID    string
	AccountID        string
	OriginalAmount   float64
	OriginalCurrency string
	AmountUSD        float64
	FeeUSD           float64
	Status           string // "APPROVED", "REVIEW_REQUIRED", "REJECTED"
	ProcessedAt      time.Time
}

// TransactionProcessor orchestrates enrichment and business rule validation.
type TransactionProcessor struct {
	feeRate float64
}

// NewTransactionProcessor creates a processor with a configured fee percentage.
func NewTransactionProcessor(feeRate float64) *TransactionProcessor {
	return &TransactionProcessor{feeRate: feeRate}
}

// Process enforces business rules, normalizes currency, and calculates processing fees.
func (tp *TransactionProcessor) Process(ctx context.Context, tx TransactionEvent) (EnrichedTransaction, error) {
	// 1. Worker panic injection test
	if tx.SimulatePanic {
		panic(fmt.Sprintf("fatal nil dereference or hardware trap processing transaction %s", tx.ID))
	}

	// 2. Transient network simulation test
	if tx.SimulateTransient {
		return EnrichedTransaction{}, fmt.Errorf("%w: connection reset by peer", ErrTransientNetwork)
	}

	// 3. Domain validation (Errors as values)
	if tx.Amount <= 0 {
		return EnrichedTransaction{}, fmt.Errorf("%w: got %f", ErrInvalidAmount, tx.Amount)
	}

	curr := strings.ToUpper(strings.TrimSpace(tx.Currency))
	rate, ok := fxRates[curr]
	if !ok {
		return EnrichedTransaction{}, fmt.Errorf("%w: currency '%s' not accepted", ErrUnsupportedCurr, curr)
	}

	if strings.HasPrefix(tx.AccountID, "SUSPENDED-") {
		return EnrichedTransaction{}, fmt.Errorf("%w: account %s", ErrAccountSuspended, tx.AccountID)
	}

	// 4. Financial calculation & FX normalization
	amountUSD := tx.Amount * rate
	feeUSD := amountUSD * tp.feeRate

	status := "APPROVED"
	if amountUSD >= 10_000.00 {
		status = "REVIEW_REQUIRED" // Anti-money laundering threshold flag
	}

	return EnrichedTransaction{
		TransactionID:    tx.ID,
		AccountID:        tx.AccountID,
		OriginalAmount:   tx.Amount,
		OriginalCurrency: curr,
		AmountUSD:        amountUSD,
		FeeUSD:           feeUSD,
		Status:           status,
		ProcessedAt:      time.Now().UTC(),
	}, nil
}
