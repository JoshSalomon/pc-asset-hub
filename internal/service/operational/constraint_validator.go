package operational

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

// CompilePatternConstraint compiles a pattern constraint from a constraints map.
// Returns (compiled regex, "") on success, (nil, error message) on bad pattern,
// or (nil, "") if no pattern constraint exists.
func CompilePatternConstraint(constraints map[string]any) (*regexp.Regexp, string) {
	if constraints == nil {
		return nil, ""
	}
	p, ok := constraints["pattern"]
	if !ok {
		return nil, ""
	}
	patternStr, ok := p.(string)
	if !ok {
		return nil, ""
	}
	// Auto-anchor: wrap in ^(?:...)$ so the pattern must match the entire string.
	// Users write patterns like [0-9A-F]+ expecting full-string match;
	// without anchoring, MatchString does partial matching (e.g., "ABCxyz" passes).
	anchored := "^(?:" + patternStr + ")$"
	compiled, err := regexp.Compile(anchored)
	if err != nil {
		return nil, fmt.Sprintf("invalid regex %q: %v", patternStr, err)
	}
	return compiled, ""
}

// ValidateValueConstraints checks an instance value against type constraints.
// Returns a slice of human-readable error strings (empty if valid).
// Takes a pre-compiled regex (nil if no pattern or pattern invalid).
func ValidateValueConstraints(
	baseType models.BaseType,
	constraints map[string]any,
	val *models.InstanceAttributeValue,
	compiledPattern *regexp.Regexp,
) []string {
	if baseType == models.BaseTypeEnum {
		return nil // enum validation handled in main loop
	}

	var errs []string

	switch baseType {
	case models.BaseTypeString:
		errs = validateString(constraints, val, compiledPattern)
	case models.BaseTypeInteger:
		errs = validateInteger(constraints, val)
	case models.BaseTypeNumber:
		errs = validateNumber(constraints, val)
	case models.BaseTypeBoolean:
		errs = validateBoolean(val)
	case models.BaseTypeDate:
		errs = validateDate(val)
	case models.BaseTypeURL:
		errs = validateURL(val)
	case models.BaseTypeJSON:
		errs = validateJSON(val)
	case models.BaseTypeList:
		errs = validateList(constraints, val)
	}

	return errs
}

func validateString(constraints map[string]any, val *models.InstanceAttributeValue, compiledPattern *regexp.Regexp) []string {
	var errs []string
	if maxLen, ok := constraintFloat64(constraints, "max_length"); ok {
		if float64(len(val.ValueString)) > maxLen {
			errs = append(errs, fmt.Sprintf("value exceeds maximum length of %d", int(maxLen)))
		}
	}
	if compiledPattern != nil && !compiledPattern.MatchString(val.ValueString) {
		patternStr, _ := constraints["pattern"].(string)
		// Show the beginning of the value so the user knows what failed
		display := val.ValueString
		if len(display) > 30 {
			display = display[:30] + "..."
		}
		errs = append(errs, fmt.Sprintf("value %q does not match pattern %q", display, patternStr))
	}
	return errs
}

func validateInteger(constraints map[string]any, val *models.InstanceAttributeValue) []string {
	if val.ValueNumber == nil {
		return nil
	}
	var errs []string
	num := *val.ValueNumber
	if num != math.Trunc(num) {
		errs = append(errs, fmt.Sprintf("value %g must be a whole number", num))
	}
	errs = append(errs, validateMinMax(constraints, num)...)
	return errs
}

func validateNumber(constraints map[string]any, val *models.InstanceAttributeValue) []string {
	if val.ValueNumber == nil {
		return nil
	}
	return validateMinMax(constraints, *val.ValueNumber)
}

func validateMinMax(constraints map[string]any, num float64) []string {
	var errs []string
	if minVal, ok := constraintFloat64(constraints, "min"); ok {
		if num < minVal {
			errs = append(errs, fmt.Sprintf("value %g is below minimum %g", num, minVal))
		}
	}
	if maxVal, ok := constraintFloat64(constraints, "max"); ok {
		if num > maxVal {
			errs = append(errs, fmt.Sprintf("value %g is above maximum %g", num, maxVal))
		}
	}
	return errs
}

func validateBoolean(val *models.InstanceAttributeValue) []string {
	if val.ValueString != "true" && val.ValueString != "false" {
		return []string{fmt.Sprintf("value %q must be \"true\" or \"false\"", val.ValueString)}
	}
	return nil
}

func validateDate(val *models.InstanceAttributeValue) []string {
	_, err1 := time.Parse("2006-01-02", val.ValueString)
	_, err2 := time.Parse(time.RFC3339, val.ValueString)
	if err1 != nil && err2 != nil {
		return []string{fmt.Sprintf("invalid date format %q: expected YYYY-MM-DD or RFC3339", val.ValueString)}
	}
	return nil
}

func validateURL(val *models.InstanceAttributeValue) []string {
	u, err := url.ParseRequestURI(val.ValueString)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return []string{fmt.Sprintf("invalid URL %q: must include scheme and host", val.ValueString)}
	}
	// Extract hostname (without port)
	host := u.Hostname()
	// Reject trailing dots (e.g., "http://127.") and empty host after port strip
	if host == "" || strings.HasSuffix(host, ".") {
		return []string{fmt.Sprintf("invalid URL %q: host is incomplete", val.ValueString)}
	}
	// If it looks like an IP, validate it
	if net.ParseIP(host) == nil {
		// Not a valid IP — check it's a plausible hostname (has at least one dot or is localhost)
		if !strings.Contains(host, ".") && host != "localhost" {
			return []string{fmt.Sprintf("invalid URL %q: host is incomplete", val.ValueString)}
		}
	}
	return nil
}

func validateJSON(val *models.InstanceAttributeValue) []string {
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(val.ValueJSON), &raw); err != nil {
		return []string{fmt.Sprintf("invalid JSON syntax: %s", err.Error())}
	}
	return nil
}

func validateList(constraints map[string]any, val *models.InstanceAttributeValue) []string {
	// Use json.Decoder with UseNumber to preserve float vs integer distinction
	dec := json.NewDecoder(strings.NewReader(val.ValueJSON))
	dec.UseNumber()
	var items []any
	if err := dec.Decode(&items); err != nil {
		return []string{fmt.Sprintf("invalid list: %v", err)}
	}

	var errs []string

	if maxLen, ok := constraintFloat64(constraints, "max_length"); ok {
		if float64(len(items)) > maxLen {
			errs = append(errs, fmt.Sprintf("list exceeds maximum length of %d (has %d)", int(maxLen), len(items)))
		}
	}

	if elemType, ok := constraints["element_base_type"].(string); ok {
		for i, item := range items {
			if !isValidElementType(elemType, item) {
				errs = append(errs, fmt.Sprintf("invalid element at index %d: expected %s", i, elemType))
			}
		}
	}

	return errs
}

func isValidElementType(elemType string, item any) bool {
	switch models.BaseType(elemType) {
	case models.BaseTypeString:
		_, ok := item.(string)
		return ok
	case models.BaseTypeNumber:
		if _, ok := item.(json.Number); ok {
			return true
		}
		_, ok := item.(float64)
		return ok
	case models.BaseTypeInteger:
		if n, ok := item.(json.Number); ok {
			// Reject float literals like "0.0" — must not contain a decimal point
			s := n.String()
			if strings.Contains(s, ".") || strings.Contains(s, "e") || strings.Contains(s, "E") {
				return false
			}
			_, err := strconv.ParseInt(s, 10, 64)
			return err == nil
		}
		f, ok := item.(float64)
		return ok && f == math.Trunc(f)
	case models.BaseTypeBoolean:
		_, ok := item.(bool)
		return ok
	default:
		return true // unknown element type: don't flag
	}
}

// constraintFloat64 extracts a numeric constraint from the constraints map.
func constraintFloat64(constraints map[string]any, key string) (float64, bool) {
	if constraints == nil {
		return 0, false
	}
	v, ok := constraints[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
