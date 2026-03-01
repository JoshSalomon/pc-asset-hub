package errors

import (
	"fmt"
	"testing"
)

func TestNewNotFound(t *testing.T) {
	err := NewNotFound("EntityType", "abc-123")
	if err.Code != "NOT_FOUND" {
		t.Fatalf("expected code NOT_FOUND, got %s", err.Code)
	}
	if !IsNotFound(err) {
		t.Fatal("IsNotFound should return true")
	}
	if err.Error() != "NOT_FOUND: EntityType not found: abc-123" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestNewConflict(t *testing.T) {
	err := NewConflict("EntityType", "name already exists")
	if !IsConflict(err) {
		t.Fatal("IsConflict should return true")
	}
	if IsNotFound(err) {
		t.Fatal("IsNotFound should return false for conflict error")
	}
}

func TestNewValidation(t *testing.T) {
	err := NewValidation("name is required")
	if !IsValidation(err) {
		t.Fatal("IsValidation should return true")
	}
}

func TestNewForbidden(t *testing.T) {
	err := NewForbidden("insufficient permissions")
	if !IsForbidden(err) {
		t.Fatal("IsForbidden should return true")
	}
}

func TestNewCycleDetected(t *testing.T) {
	err := NewCycleDetected("A -> B -> A")
	if !IsCycleDetected(err) {
		t.Fatal("IsCycleDetected should return true")
	}
}

func TestNewReferencedEnum(t *testing.T) {
	err := NewReferencedEnum("Status", []string{"Model.status", "Tool.status"})
	if !IsReferencedEnum(err) {
		t.Fatal("IsReferencedEnum should return true")
	}
}

func TestDomainErrorWithWrappedError(t *testing.T) {
	inner := fmt.Errorf("database connection failed")
	err := &DomainError{
		Code:    "NOT_FOUND",
		Message: "entity not found",
		Err:     inner,
	}
	if err.Unwrap() != inner {
		t.Fatal("Unwrap should return the inner error")
	}
	expected := "NOT_FOUND: entity not found: database connection failed"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestNewDeepCopyRequired(t *testing.T) {
	err := NewDeepCopyRequired("version is pinned in a non-development catalog")
	if err.Code != "DEEP_COPY_REQUIRED" {
		t.Fatalf("expected code DEEP_COPY_REQUIRED, got %s", err.Code)
	}
	if err.Message != "version is pinned in a non-development catalog" {
		t.Fatalf("unexpected message: %s", err.Message)
	}
}

func TestIsDeepCopyRequired_True(t *testing.T) {
	err := NewDeepCopyRequired("version is pinned")
	if !IsDeepCopyRequired(err) {
		t.Fatal("IsDeepCopyRequired should return true for DeepCopyRequired error")
	}
}

func TestIsDeepCopyRequired_False(t *testing.T) {
	err := NewConflict("EntityType", "name already exists")
	if IsDeepCopyRequired(err) {
		t.Fatal("IsDeepCopyRequired should return false for non-DeepCopyRequired error")
	}
}

func TestIsCheckersReturnFalseForNonDomainErrors(t *testing.T) {
	err := fmt.Errorf("some other error")
	if IsNotFound(err) {
		t.Fatal("IsNotFound should return false for non-domain error")
	}
	if IsConflict(err) {
		t.Fatal("IsConflict should return false for non-domain error")
	}
	if IsValidation(err) {
		t.Fatal("IsValidation should return false for non-domain error")
	}
	if IsForbidden(err) {
		t.Fatal("IsForbidden should return false for non-domain error")
	}
	if IsCycleDetected(err) {
		t.Fatal("IsCycleDetected should return false for non-domain error")
	}
	if IsReferencedEnum(err) {
		t.Fatal("IsReferencedEnum should return false for non-domain error")
	}
	if IsDeepCopyRequired(err) {
		t.Fatal("IsDeepCopyRequired should return false for non-domain error")
	}
}
