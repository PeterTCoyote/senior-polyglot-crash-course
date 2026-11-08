package service_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	service "senior-polyglot-crash-course/day-06-testing-and-benchmarking/go"
)

// ============================================================================
// SENIOR ARCHITECTURAL LESSON: Idiomatic Go Fakes vs Java Mockito
// ============================================================================
// In Java, developers routinely use Mockito:
//   @Mock PaymentGateway gateway;
//   when(gateway.charge(any(), any(), any())).thenReturn(new Receipt(...));
//   verify(gateway, times(1)).charge(...);
//
// Mockito uses runtime bytecode generation (ByteBuddy/CGLIB) and dynamic proxies.
// Go has NO runtime bytecode manipulation, NO dynamic classloading, and NO annotations.
//
// Instead, Go relies on:
//   1. Structural Subtyping: Any struct satisfying the interface implicitly implements it.
//   2. Function-Field Stubs: A stub struct whose methods delegate to function fields.
//   3. In-memory Fakes: Lightweight real implementations (e.g. a map with a mutex).
// ============================================================================

// stubGateway implements service.PaymentGateway by delegating to a function field.
// This allows individual tests to customize behavior per-test without dynamic mock frameworks.
type stubGateway struct {
	chargeFn func(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error)
	calls    atomic.Int32
}

func (s *stubGateway) Charge(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error) {
	s.calls.Add(1)
	if s.chargeFn != nil {
		return s.chargeFn(ctx, orderID, amountCents, currency)
	}
	return nil, errors.New("chargeFn not configured on stubGateway")
}

func (s *stubGateway) CallCount() int {
	return int(s.calls.Load())
}

// TestProcessPayment_SuccessfulSettlement demonstrates a happy-path scenario with a stub.
func TestProcessPayment_SuccessfulSettlement(t *testing.T) {
	expectedReceipt := &service.Receipt{
		TransactionID: "txn-999",
		OrderID:       "ord-100",
		AmountCharged: 5000,
		Currency:      "USD",
		ProcessedAt:   time.Now().UTC(),
	}

	gatewayStub := &stubGateway{
		chargeFn: func(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error) {
			if orderID != "ord-100" || amountCents != 5000 || currency != "USD" {
				t.Errorf("unexpected gateway invocation arguments: id=%s amount=%d curr=%s", orderID, amountCents, currency)
			}
			return expectedReceipt, nil
		},
	}

	store := service.NewSafeMemoryStore()
	svc := service.NewOrderService(gatewayStub, store)

	order := service.Order{
		ID:          "ord-100",
		CustomerID:  "cust-1",
		AmountCents: 5000,
		Currency:    "USD",
	}

	receipt, err := svc.ProcessPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected payment failure: %v", err)
	}

	if receipt.TransactionID != expectedReceipt.TransactionID {
		t.Errorf("expected transaction ID %q, got %q", expectedReceipt.TransactionID, receipt.TransactionID)
	}

	// Verify gateway was called exactly once (equivalent to Mockito verify(gateway, times(1)))
	if count := gatewayStub.CallCount(); count != 1 {
		t.Errorf("expected gateway call count 1, got %d", count)
	}
}

// TestProcessPayment_IdempotencyHit verifies that an existing receipt bypasses downstream charging.
func TestProcessPayment_IdempotencyHit(t *testing.T) {
	existingReceipt := &service.Receipt{
		TransactionID: "txn-cached-1",
		OrderID:       "ord-cached-100",
		AmountCharged: 2500,
		Currency:      "EUR",
		ProcessedAt:   time.Now().Add(-10 * time.Minute),
	}

	store := service.NewSafeMemoryStore()
	if err := store.Put("ord-cached-100", existingReceipt); err != nil {
		t.Fatalf("failed pre-seeding store: %v", err)
	}

	gatewayStub := &stubGateway{
		chargeFn: func(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error) {
			t.Fatal("CRITICAL: Gateway was called for an order that should have hit idempotency cache!")
			return nil, nil
		},
	}

	svc := service.NewOrderService(gatewayStub, store)

	order := service.Order{
		ID:          "ord-cached-100",
		CustomerID:  "cust-cached",
		AmountCents: 2500,
		Currency:    "EUR",
	}

	receipt, err := svc.ProcessPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receipt.TransactionID != "txn-cached-1" {
		t.Errorf("expected cached receipt, got %+v", receipt)
	}

	// Mockito equivalent: verifyNoInteractions(gateway);
	if count := gatewayStub.CallCount(); count != 0 {
		t.Errorf("expected 0 gateway calls, got %d", count)
	}
}

// TestProcessPayment_GatewayFailure verifies error propagation when downstream call fails.
func TestProcessPayment_GatewayFailure(t *testing.T) {
	gatewayStub := &stubGateway{
		chargeFn: func(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error) {
			return nil, service.ErrGatewayUnavailable
		},
	}

	store := service.NewSafeMemoryStore()
	svc := service.NewOrderService(gatewayStub, store)

	order := service.Order{
		ID:          "ord-fail",
		CustomerID:  "cust-fail",
		AmountCents: 1000,
		Currency:    "GBP",
	}

	_, err := svc.ProcessPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, service.ErrGatewayUnavailable) {
		t.Errorf("expected error wrapping ErrGatewayUnavailable, got: %v", err)
	}
}
