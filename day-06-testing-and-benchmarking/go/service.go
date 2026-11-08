// Package service demonstrates core business logic for order processing and payment verification.
// It is intentionally structured to illustrate table-driven testing, mocking via structural interfaces,
// data race detection, and microbenchmarking for senior developers transitioning from Java.
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Sentinel errors for explicit domain failure paths.
// Unlike Java's checked/unchecked exception hierarchies, Go treats errors as values.
var (
	ErrInvalidAmount       = errors.New("invalid order amount: must be strictly positive")
	ErrUnsupportedCurrency = errors.New("unsupported currency: only USD, EUR, and GBP are accepted")
	ErrEmptyOrderID        = errors.New("order ID cannot be empty")
	ErrEmptyCustomerID     = errors.New("customer ID cannot be empty")
	ErrDuplicateOrder      = errors.New("order has already been processed (idempotency conflict)")
	ErrGatewayUnavailable  = errors.New("upstream payment gateway unavailable")
)

// SupportedCurrencies maps allowed ISO-4217 currency codes.
var SupportedCurrencies = map[string]struct{}{
	"USD": {},
	"EUR": {},
	"GBP": {},
}

// Order represents the incoming checkout transaction request.
type Order struct {
	ID          string
	CustomerID  string
	AmountCents int64
	Currency    string
	Metadata    map[string]string
}

// Receipt is returned upon successful payment settlement.
type Receipt struct {
	TransactionID string
	OrderID       string
	AmountCharged int64
	Currency      string
	ProcessedAt   time.Time
}

// PaymentGateway defines the contract for communicating with downstream merchant payment providers.
// In Java, this would be an interface implemented by a Spring @Component and mocked by Mockito.
// In Go, interfaces should be small (often 1-3 methods) and defined by the consumer, not the producer.
type PaymentGateway interface {
	Charge(ctx context.Context, orderID string, amountCents int64, currency string) (*Receipt, error)
}

// IdempotencyStore defines the storage contract for preventing duplicate payment charges.
type IdempotencyStore interface {
	Get(orderID string) (*Receipt, bool)
	Put(orderID string, receipt *Receipt) error
}

// UnsafeMemoryStore is an intentionally thread-UNSAFE store used to demonstrate how
// `go test -race` detects concurrent data races that would slip through standard unit tests.
// Java analogy: A plain java.util.HashMap shared across threads without synchronization.
type UnsafeMemoryStore struct {
	records map[string]*Receipt
}

// NewUnsafeMemoryStore initializes an unsynchronized memory store.
func NewUnsafeMemoryStore() *UnsafeMemoryStore {
	return &UnsafeMemoryStore{
		records: make(map[string]*Receipt),
	}
}

func (s *UnsafeMemoryStore) Get(orderID string) (*Receipt, bool) {
	r, exists := s.records[orderID]
	return r, exists
}

func (s *UnsafeMemoryStore) Put(orderID string, receipt *Receipt) error {
	s.records[orderID] = receipt
	return nil
}

// SafeMemoryStore is a thread-safe implementation protected by a reader-writer mutex.
// Java analogy: java.util.concurrent.ConcurrentHashMap or ReadWriteLock guarding a HashMap.
type SafeMemoryStore struct {
	mu      sync.RWMutex
	records map[string]*Receipt
}

// NewSafeMemoryStore initializes a mutex-guarded memory store.
func NewSafeMemoryStore() *SafeMemoryStore {
	return &SafeMemoryStore{
		records: make(map[string]*Receipt),
	}
}

func (s *SafeMemoryStore) Get(orderID string) (*Receipt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, exists := s.records[orderID]
	return r, exists
}

func (s *SafeMemoryStore) Put(orderID string, receipt *Receipt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[orderID]; exists {
		return ErrDuplicateOrder
	}
	s.records[orderID] = receipt
	return nil
}

// OrderService orchestrates payment verification, validation, and settlement.
type OrderService struct {
	gateway PaymentGateway
	store   IdempotencyStore
}

// NewOrderService constructs a service instance with injected dependencies.
// Senior Note: Go constructors are plain factory functions returning structs or pointers.
// No dependency injection container (Spring / Guice) is required or idiomatic.
func NewOrderService(gateway PaymentGateway, store IdempotencyStore) *OrderService {
	return &OrderService{
		gateway: gateway,
		store:   store,
	}
}

// ValidateOrder performs stateless business invariant checks on incoming orders.
// This is an ideal candidate for Table-Driven Tests.
func (s *OrderService) ValidateOrder(order Order) error {
	if order.ID == "" {
		return ErrEmptyOrderID
	}
	if order.CustomerID == "" {
		return ErrEmptyCustomerID
	}
	if order.AmountCents <= 0 {
		return ErrInvalidAmount
	}
	if _, ok := SupportedCurrencies[order.Currency]; !ok {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, order.Currency)
	}
	return nil
}

// ProcessPayment validates the order, verifies idempotency, charges the gateway,
// and saves the receipt.
func (s *OrderService) ProcessPayment(ctx context.Context, order Order) (*Receipt, error) {
	// 1. Invariant Validation
	if err := s.ValidateOrder(order); err != nil {
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	// 2. Check Idempotency Store
	if existing, found := s.store.Get(order.ID); found {
		return existing, nil
	}

	// 3. Upstream Gateway Call
	receipt, err := s.gateway.Charge(ctx, order.ID, order.AmountCents, order.Currency)
	if err != nil {
		return nil, fmt.Errorf("payment gateway rejected transaction: %w", err)
	}

	// 4. Record to Idempotency Store
	if err := s.store.Put(order.ID, receipt); err != nil {
		// SENIOR CONCURRENCY PATTERN: If another concurrent worker inserted the receipt
		// during our upstream call, fetch the existing receipt rather than failing.
		if errors.Is(err, ErrDuplicateOrder) {
			if existing, found := s.store.Get(order.ID); found {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("idempotency recording failed: %w", err)
	}

	return receipt, nil
}
