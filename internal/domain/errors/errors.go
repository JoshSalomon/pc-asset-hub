package errors

import "fmt"

type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewNotFound(entity string, id string) *DomainError {
	return &DomainError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found: %s", entity, id),
	}
}

func NewConflict(entity string, detail string) *DomainError {
	return &DomainError{
		Code:    "CONFLICT",
		Message: fmt.Sprintf("%s conflict: %s", entity, detail),
	}
}

func NewValidation(detail string) *DomainError {
	return &DomainError{
		Code:    "VALIDATION",
		Message: detail,
	}
}

func NewForbidden(detail string) *DomainError {
	return &DomainError{
		Code:    "FORBIDDEN",
		Message: detail,
	}
}

func NewCycleDetected(detail string) *DomainError {
	return &DomainError{
		Code:    "CYCLE_DETECTED",
		Message: detail,
	}
}

func NewReferencedEnum(enumName string, references []string) *DomainError {
	return &DomainError{
		Code:    "REFERENCED_ENUM",
		Message: fmt.Sprintf("enum %q is referenced by: %v", enumName, references),
	}
}

func IsNotFound(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "NOT_FOUND"
	}
	return false
}

func IsConflict(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "CONFLICT"
	}
	return false
}

func IsValidation(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "VALIDATION"
	}
	return false
}

func IsForbidden(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "FORBIDDEN"
	}
	return false
}

func IsCycleDetected(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "CYCLE_DETECTED"
	}
	return false
}

func IsReferencedEnum(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "REFERENCED_ENUM"
	}
	return false
}

func NewDeepCopyRequired(detail string) *DomainError {
	return &DomainError{
		Code:    "DEEP_COPY_REQUIRED",
		Message: detail,
	}
}

func IsDeepCopyRequired(err error) bool {
	if e, ok := err.(*DomainError); ok {
		return e.Code == "DEEP_COPY_REQUIRED"
	}
	return false
}
