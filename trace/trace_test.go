package trace

import (
	"context"
	"testing"
)

func TestGenerateTraceID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()
	
	if id1 == "" {
		t.Error("GenerateTraceID returned empty string")
	}
	
	if id1 == id2 {
		t.Error("GenerateTraceID returned duplicate IDs")
	}
	
	if len(id1) != 32 {
		t.Errorf("GenerateTraceID returned ID with wrong length: got %d, want 32", len(id1))
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-id-12345"
	
	ctx = WithTraceID(ctx, traceID)
	
	retrieved := GetTraceID(ctx)
	if retrieved != traceID {
		t.Errorf("GetTraceID returned wrong ID: got %s, want %s", retrieved, traceID)
	}
}

func TestGetTraceID_NoID(t *testing.T) {
	ctx := context.Background()
	
	id := GetTraceID(ctx)
	if id != "" {
		t.Errorf("GetTraceID should return empty string when no ID set, got %s", id)
	}
}
