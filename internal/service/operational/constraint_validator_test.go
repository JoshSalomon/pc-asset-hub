package operational

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

// === CompilePatternConstraint tests ===

func TestCompilePatternConstraint(t *testing.T) {
	tests := []struct {
		name        string
		constraints map[string]any
		wantNil     bool   // expect compiled to be nil
		wantErrMsg  bool   // expect non-empty error message
		matchStr    string // if compiled non-nil, test this string matches
		noMatchStr  string // if compiled non-nil, test this string does not match
	}{
		{
			name:        "no pattern key",
			constraints: map[string]any{"max_length": 10},
			wantNil:     true,
		},
		{
			name:        "valid pattern",
			constraints: map[string]any{"pattern": "^[a-z]+$"},
			wantNil:     false,
			matchStr:    "abc",
			noMatchStr:  "ABC",
		},
		{
			name:        "invalid pattern",
			constraints: map[string]any{"pattern": "[invalid"},
			wantNil:     true,
			wantErrMsg:  true,
		},
		{
			name:        "non-string pattern value",
			constraints: map[string]any{"pattern": 123},
			wantNil:     true,
		},
		{
			name:        "nil constraints",
			constraints: nil,
			wantNil:     true,
		},
		{
			name:        "unanchored pattern gets auto-anchored",
			constraints: map[string]any{"pattern": "[0-9A-F]+"},
			wantNil:     false,
			matchStr:    "ABCDEF",
			noMatchStr:  "ABCxyz", // partial match must NOT pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, errMsg := CompilePatternConstraint(tt.constraints)
			if tt.wantNil {
				assert.Nil(t, compiled)
			} else {
				assert.NotNil(t, compiled)
			}
			if tt.wantErrMsg {
				assert.NotEmpty(t, errMsg)
			} else {
				assert.Empty(t, errMsg)
			}
			if compiled != nil && tt.matchStr != "" {
				assert.True(t, compiled.MatchString(tt.matchStr))
			}
			if compiled != nil && tt.noMatchStr != "" {
				assert.False(t, compiled.MatchString(tt.noMatchStr))
			}
		})
	}
}

// === ValidateValueConstraints — table-driven tests ===

func TestValidateValueConstraints(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name        string
		baseType    models.BaseType
		constraints map[string]any
		val         *models.InstanceAttributeValue
		pattern     *regexp.Regexp
		wantErrors  int
		wantContain string // substring expected in at least one error
	}{
		// T-31.47: String exceeding max_length → invalid
		{
			name:        "string exceeds max_length",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"max_length": float64(5)},
			val:         &models.InstanceAttributeValue{ValueString: "toolong"},
			wantErrors:  1,
			wantContain: "exceeds maximum length",
		},
		// T-31.49: String within constraints → valid
		{
			name:        "string within max_length",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"max_length": float64(10)},
			val:         &models.InstanceAttributeValue{ValueString: "short"},
			wantErrors:  0,
		},
		// T-31.48: String not matching pattern → invalid
		{
			name:        "string fails pattern",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"pattern": "^[a-z]+$"},
			val:         &models.InstanceAttributeValue{ValueString: "ABC123"},
			pattern:     regexp.MustCompile("^[a-z]+$"),
			wantErrors:  1,
			wantContain: "does not match pattern",
		},
		// String matches pattern → valid
		{
			name:        "string matches pattern",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"pattern": "^[a-z]+$"},
			val:         &models.InstanceAttributeValue{ValueString: "abc"},
			pattern:     regexp.MustCompile("^[a-z]+$"),
			wantErrors:  0,
		},
		// Pattern error message includes the failing value
		{
			name:        "pattern error shows value",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"pattern": "[0-9A-F]+"},
			val:         &models.InstanceAttributeValue{ValueString: "xyz"},
			pattern:     regexp.MustCompile("^(?:[0-9A-F]+)$"),
			wantErrors:  1,
			wantContain: `"xyz" does not match`,
		},
		// Pattern error truncates long values
		{
			name:        "pattern error truncates long value",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"pattern": "[0-9]+"},
			val:         &models.InstanceAttributeValue{ValueString: "this-is-a-very-long-string-that-exceeds-thirty-characters-easily"},
			pattern:     regexp.MustCompile("^(?:[0-9]+)$"),
			wantErrors:  1,
			wantContain: "...",
		},
		// Unanchored pattern: partial match must NOT pass
		{
			name:        "unanchored pattern rejects partial match",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"pattern": "[0-9A-F]+"},
			val:         &models.InstanceAttributeValue{ValueString: "ABCxyz"},
			pattern:     regexp.MustCompile("^(?:[0-9A-F]+)$"), // anchored by CompilePatternConstraint
			wantErrors:  1,
			wantContain: "does not match pattern",
		},
		{
			name:        "unanchored pattern accepts full match",
			baseType:    models.BaseTypeString,
			constraints: map[string]any{"pattern": "[0-9A-F]+"},
			val:         &models.InstanceAttributeValue{ValueString: "ABCDEF"},
			pattern:     regexp.MustCompile("^(?:[0-9A-F]+)$"),
			wantErrors:  0,
		},
		// T-31.50: Integer not whole number → invalid
		{
			name:        "integer not whole number",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(3.14)},
			wantErrors:  1,
			wantContain: "must be a whole number",
		},
		// T-31.51: Integer below min → invalid
		{
			name:        "integer below min",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{"min": float64(10)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(5)},
			wantErrors:  1,
			wantContain: "below minimum",
		},
		// T-31.52: Integer above max → invalid
		{
			name:        "integer above max",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{"max": float64(100)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(200)},
			wantErrors:  1,
			wantContain: "above maximum",
		},
		// T-31.53: Integer within range → valid
		{
			name:        "integer within range",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{"min": float64(1), "max": float64(100)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(50)},
			wantErrors:  0,
		},
		// T-31.54: Number below min → invalid
		{
			name:        "number below min",
			baseType:    models.BaseTypeNumber,
			constraints: map[string]any{"min": float64(0)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(-5.5)},
			wantErrors:  1,
			wantContain: "below minimum",
		},
		// T-31.55: Number above max → invalid
		{
			name:        "number above max",
			baseType:    models.BaseTypeNumber,
			constraints: map[string]any{"max": float64(99.9)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(100.0)},
			wantErrors:  1,
			wantContain: "above maximum",
		},
		// Number within range → valid
		{
			name:        "number within range",
			baseType:    models.BaseTypeNumber,
			constraints: map[string]any{"min": float64(0), "max": float64(100)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(50.5)},
			wantErrors:  0,
		},
		// T-31.56: Boolean not "true"/"false" → invalid
		{
			name:        "boolean invalid value",
			baseType:    models.BaseTypeBoolean,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "yes"},
			wantErrors:  1,
			wantContain: "must be \"true\" or \"false\"",
		},
		// Boolean valid → valid
		{
			name:        "boolean true valid",
			baseType:    models.BaseTypeBoolean,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "true"},
			wantErrors:  0,
		},
		{
			name:        "boolean false valid",
			baseType:    models.BaseTypeBoolean,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "false"},
			wantErrors:  0,
		},
		// T-31.57: Date not ISO 8601 → invalid
		{
			name:        "date invalid format",
			baseType:    models.BaseTypeDate,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "not-a-date"},
			wantErrors:  1,
			wantContain: "invalid date format",
		},
		// Date valid ISO → valid
		{
			name:        "date valid YYYY-MM-DD",
			baseType:    models.BaseTypeDate,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "2026-04-15"},
			wantErrors:  0,
		},
		// Date valid RFC3339 → valid
		{
			name:        "date valid RFC3339",
			baseType:    models.BaseTypeDate,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "2026-04-15T10:30:00Z"},
			wantErrors:  0,
		},
		// T-31.58: URL not valid → invalid
		{
			name:        "url invalid",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "not a url"},
			wantErrors:  1,
			wantContain: "invalid URL",
		},
		// URL missing scheme → invalid
		{
			name:        "url missing scheme",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "example.com"},
			wantErrors:  1,
			wantContain: "invalid URL",
		},
		// URL valid → valid
		{
			name:        "url valid https",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "https://example.com/path"},
			wantErrors:  0,
		},
		// URL with trailing dot in host → invalid
		{
			name:        "url trailing dot in host",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "http://127."},
			wantErrors:  1,
			wantContain: "host is incomplete",
		},
		// URL with bare number host → invalid
		{
			name:        "url bare number host",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "http://127"},
			wantErrors:  1,
			wantContain: "host is incomplete",
		},
		// URL with valid IP → valid
		{
			name:        "url valid ip",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "http://127.0.0.1:8080/api"},
			wantErrors:  0,
		},
		// URL with localhost → valid
		{
			name:        "url localhost",
			baseType:    models.BaseTypeURL,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueString: "http://localhost:3000"},
			wantErrors:  0,
		},
		// T-31.64: JSON not valid syntax → invalid
		{
			name:        "json invalid syntax",
			baseType:    models.BaseTypeJSON,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueJSON: "{not json}"},
			wantErrors:  1,
			wantContain: "invalid JSON",
		},
		// JSON error message includes position detail from json.Unmarshal
		{
			name:        "json error includes detail",
			baseType:    models.BaseTypeJSON,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueJSON: `{"key": bad}`},
			wantErrors:  1,
			wantContain: "invalid character",
		},
		// T-31.65: JSON valid → valid
		{
			name:        "json valid",
			baseType:    models.BaseTypeJSON,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueJSON: `{"key":"value"}`},
			wantErrors:  0,
		},
		// T-31.61: List exceeds max_length → invalid
		{
			name:        "list exceeds max_length",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"max_length": float64(2)},
			val:         &models.InstanceAttributeValue{ValueJSON: `["a","b","c"]`},
			wantErrors:  1,
			wantContain: "exceeds maximum length",
		},
		// T-31.63: List with valid elements → valid
		{
			name:        "list valid",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"max_length": float64(5)},
			val:         &models.InstanceAttributeValue{ValueJSON: `["a","b"]`},
			wantErrors:  0,
		},
		// T-31.62: List element wrong type → invalid
		{
			name:        "list element wrong type",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "integer"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[1, "not-an-int", 3]`},
			wantErrors:  1,
			wantContain: "invalid element",
		},
		// List valid elements with type check
		{
			name:        "list elements match type",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "number"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[1.5, 2.3, 99]`},
			wantErrors:  0,
		},
		// List invalid JSON → invalid
		{
			name:        "list invalid json",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{},
			val:         &models.InstanceAttributeValue{ValueJSON: "not json"},
			wantErrors:  1,
			wantContain: "invalid list",
		},
		// Enum → skip (handled in main loop)
		{
			name:        "enum skipped",
			baseType:    models.BaseTypeEnum,
			constraints: map[string]any{"values": []any{"a", "b"}},
			val:         &models.InstanceAttributeValue{ValueString: "c"},
			wantErrors:  0, // enum validation handled elsewhere
		},
		// No constraints → no errors
		{
			name:        "no constraints no errors",
			baseType:    models.BaseTypeString,
			constraints: nil,
			val:         &models.InstanceAttributeValue{ValueString: "anything"},
			wantErrors:  0,
		},
		// List element type = string, non-string element → invalid
		{
			name:        "list element wrong type string",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "string"},
			val:         &models.InstanceAttributeValue{ValueJSON: `["ok", 42]`},
			wantErrors:  1,
			wantContain: "invalid element at index 1",
		},
		// List element type = string, all strings → valid
		{
			name:        "list element type string valid",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "string"},
			val:         &models.InstanceAttributeValue{ValueJSON: `["a","b","c"]`},
			wantErrors:  0,
		},
		// List element type = boolean, non-boolean → invalid
		{
			name:        "list element wrong type boolean",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "boolean"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[true, "nope"]`},
			wantErrors:  1,
			wantContain: "invalid element at index 1",
		},
		// List element type = boolean, all booleans → valid
		{
			name:        "list element type boolean valid",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "boolean"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[true, false, true]`},
			wantErrors:  0,
		},
		// TD-116: List with 0.0 float literal should fail integer validation
		{
			name:        "list integer rejects float literal 0.0",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "integer"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[1, 2, 0.0, 4]`},
			wantErrors:  1,
			wantContain: "invalid element at index 2",
		},
		// List with whole numbers as integers is valid
		{
			name:        "list integer accepts whole numbers",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "integer"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[1, 2, 0, 4]`},
			wantErrors:  0,
		},
		// List element type = unknown → no validation (all pass)
		{
			name:        "list element type unknown passes all",
			baseType:    models.BaseTypeList,
			constraints: map[string]any{"element_base_type": "unknown_type"},
			val:         &models.InstanceAttributeValue{ValueJSON: `[1, "two", true]`},
			wantErrors:  0,
		},
		// Integer nil value number → no panic (edge case)
		{
			name:        "integer nil value number",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{"min": float64(0)},
			val:         &models.InstanceAttributeValue{ValueNumber: nil},
			wantErrors:  0, // empty values handled before calling this function
		},
		// Number nil value number → no panic
		{
			name:        "number nil value number",
			baseType:    models.BaseTypeNumber,
			constraints: map[string]any{"min": float64(0)},
			val:         &models.InstanceAttributeValue{ValueNumber: nil},
			wantErrors:  0,
		},
		// constraintFloat64: int value (Go map literals with untyped int → int)
		{
			name:        "integer constraint as int type",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{"min": int(5)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(3)},
			wantErrors:  1,
			wantContain: "below minimum",
		},
		// constraintFloat64: int64 value
		{
			name:        "integer constraint as int64 type",
			baseType:    models.BaseTypeInteger,
			constraints: map[string]any{"max": int64(10)},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(20)},
			wantErrors:  1,
			wantContain: "above maximum",
		},
		// constraintFloat64: non-numeric value → ignored (no error, no constraint applied)
		{
			name:        "non-numeric constraint value ignored",
			baseType:    models.BaseTypeNumber,
			constraints: map[string]any{"min": "not-a-number"},
			val:         &models.InstanceAttributeValue{ValueNumber: floatPtr(-100)},
			wantErrors:  0, // constraint ignored, no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateValueConstraints(tt.baseType, tt.constraints, tt.val, tt.pattern)
			assert.Len(t, errs, tt.wantErrors, "error count mismatch")
			if tt.wantContain != "" && len(errs) > 0 {
				found := false
				for _, e := range errs {
					if strings.Contains(e, tt.wantContain) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error containing %q, got %v", tt.wantContain, errs)
			}
		})
	}
}

// === isValidElementType direct tests (float64 fallback paths) ===

func TestIsValidElementType_Float64Fallback(t *testing.T) {
	// When items come as float64 (e.g., from json.Unmarshal without UseNumber),
	// the float64 fallback branches in isValidElementType should work correctly.

	// BaseTypeNumber: float64 → valid
	assert.True(t, isValidElementType("number", float64(3.14)))

	// BaseTypeNumber: string → invalid
	assert.False(t, isValidElementType("number", "not-a-number"))

	// BaseTypeInteger: float64 whole number → valid
	assert.True(t, isValidElementType("integer", float64(42)))

	// BaseTypeInteger: float64 with fraction → invalid
	assert.False(t, isValidElementType("integer", float64(3.5)))
}

