package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	breaker := New(3, 100*time.Millisecond)
	
	// Should succeed initially
	err := breaker.Call(func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Call should succeed in closed state, got error: %v", err)
	}
}

func TestCircuitBreaker_OpenState(t *testing.T) {
	breaker := New(3, 100*time.Millisecond)
	
	// Trigger 3 failures to open circuit
	for i := 0; i < 3; i++ {
		breaker.Call(func() error {
			return errors.New("test error")
		})
	}
	
	// Next call should fail fast
	err := breaker.Call(func() error {
		t.Error("Function should not be called when circuit is open")
		return nil
	})
	
	if err == nil {
		t.Error("Call should fail when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	breaker := New(2, 50*time.Millisecond)
	
	// Open the circuit
	for i := 0; i < 2; i++ {
		breaker.Call(func() error {
			return errors.New("test error")
		})
	}
	
	// Wait for timeout
	time.Sleep(60 * time.Millisecond)
	
	// Should enter half-open state and allow one probe
	called := false
	err := breaker.Call(func() error {
		called = true
		return nil
	})
	
	if !called {
		t.Error("Function should be called in half-open state")
	}
	
	if err != nil {
		t.Errorf("Call should succeed in half-open state, got error: %v", err)
	}
}
