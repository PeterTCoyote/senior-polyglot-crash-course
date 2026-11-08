// Package service_test provides isolated external test coverage for the service package.
// Notice the package name `service_test`: in Go, this enforces the "Black-Box Testing" pattern,
// ensuring the test can only access exported members, preventing tests from coupling to private state.
package service_test

import (
	"errors"
	"testing"

	service "senior-polyglot-crash-course/day-06-testing-and-benchmarking/go"
)

// TestValidateOrder_TableDriven demonstrates idiomatic Go Table-Driven Tests.
//
// JAVA TRANSLATION NOTE:
// In JUnit 5, you would write:
//
//   @ParameterizedTest(name = "{0}")
//   @MethodSource("provideValidationCases")
//   void testValidateOrder(String name, Order order, Class<? extends Exception> expectedEx) { ... }
//
// In Go, table-driven tests are built using an anonymous slice of structs.
// There is zero reflection, zero annotations, and full compile-time type safety.
func TestValidateOrder_TableDriven(t *testing.T) {
	// 1. Define the test matrix as a slice of anonymous structs.
	tests := []struct {
		name        string
		order       service.Order
		expectedErr error
	}{
		{
			name: "valid USD transaction",
			order: service.Order{
				ID:          "ord-001",
				CustomerID:  "cust-101",
				AmountCents: 5000,
				Currency:    "USD",
			},
			expectedErr: nil,
		},
		{
			name: "valid EUR transaction",
			order: service.Order{
				ID:          "ord-002",
				CustomerID:  "cust-102",
				AmountCents: 1250,
				Currency:    "EUR",
			},
			expectedErr: nil,
		},
		{
			name: "missing order ID",
			order: service.Order{
				ID:          "",
				CustomerID:  "cust-103",
				AmountCents: 2000,
				Currency:    "USD",
			},
			expectedErr: service.ErrEmptyOrderID,
		},
		{
			name: "missing customer ID",
			order: service.Order{
				ID:          "ord-004",
				CustomerID:  "",
				AmountCents: 3500,
				Currency:    "USD",
			},
			expectedErr: service.ErrEmptyCustomerID,
		},
		{
			name: "zero amount rejected",
			order: service.Order{
				ID:          "ord-005",
				CustomerID:  "cust-105",
				AmountCents: 0,
				Currency:    "USD",
			},
			expectedErr: service.ErrInvalidAmount,
		},
		{
			name: "negative amount rejected",
			order: service.Order{
				ID:          "ord-006",
				CustomerID:  "cust-106",
				AmountCents: -500,
				Currency:    "USD",
			},
			expectedErr: service.ErrInvalidAmount,
		},
		{
			name: "unsupported currency code (JPY)",
			order: service.Order{
				ID:          "ord-007",
				CustomerID:  "cust-107",
				AmountCents: 10000,
				Currency:    "JPY",
			},
			expectedErr: service.ErrUnsupportedCurrency,
		},
	}

	// Instantiate the unit under test (no mocks needed for pure invariant validation).
	svc := service.NewOrderService(nil, nil)

	// 2. Iterate through test cases using t.Run() for discrete subtest reporting.
	for _, tc := range tests {
		// SENIOR GOTCHA - LOOP VARIABLE CAPTURE:
		// Prior to Go 1.22, running `t.Parallel()` inside this loop without `tc := tc`
		// caused a notorious data race where all goroutines captured the single shared `tc` variable.
		// In Go 1.22+ and Go 1.27, for-loop variables have per-iteration scope by default.
		// We can safely call t.Parallel() directly!
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Enables parallel execution across logical CPU cores.

			err := svc.ValidateOrder(tc.order)

			// SENIOR TRANSLATION: Assertions in Go vs Java
			// In Java, an assertion failure throws `AssertionError` which immediately terminates the method.
			// In Go:
			// - `t.Errorf()` records failure, formats diagnostic message, but CONTINUES execution
			//   (equivalent to AssertJ's `SoftAssertions`).
			// - `t.Fatalf()` prints failure and halts the CURRENT subtest goroutine immediately.
			// Always use errors.Is() to unpack wrapped sentinel errors (%w).
			if tc.expectedErr != nil {
				if err == nil {
					t.Fatalf("expected error [%v], got nil", tc.expectedErr)
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected error wrapping [%v], got [%v]", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected valid order, got unexpected error: %v", err)
				}
			}
		})
	}
}
