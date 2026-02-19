package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"unicode"
)

var reAmount = regexp.MustCompile(`^\d{1,12}([.,]\d{1,2})?$`)
var reAmountEURDot = regexp.MustCompile(`^\d{1,12}(\.\d{1,2})?$`)
var reAmountEURComma = regexp.MustCompile(`^\d{1,12}(,\d{1,2})?$`)
var reAmountEURGroupedSpaceComma = regexp.MustCompile(`^\d{1,3}( \d{3})*(,\d{1,2})?$`)
var reAmountEURGroupedDotComma = regexp.MustCompile(`^\d{1,3}(\.\d{3})*(,\d{1,2})?$`)
var amountLenientOCR atomic.Bool

func SetAmountLenientOCR(enabled bool) {
	amountLenientOCR.Store(enabled)
}

func parseAmountEUR(s, amountFormat string) (int64, error) {
	v := strings.TrimSpace(s)
	if v == "" {
		return 0, fmt.Errorf("amount is required")
	}
	format := strings.ToLower(strings.TrimSpace(amountFormat))

	var (
		normalized string
		currency   string
		err        error
	)
	switch format {
	case "", "auto":
		if amountLenientOCR.Load() {
			normalized, currency, err = normalizeAmountInputLenient(v)
		} else {
			normalized, currency, err = normalizeAmountInput(v)
		}
	case "auto_eur_lenient":
		if !amountLenientOCR.Load() {
			return 0, fmt.Errorf("unsupported amount_format")
		}
		normalized, currency, err = normalizeAmountInputLenient(v)
	case "eur_dot", "eur_comma", "eur_grouped_space_comma", "eur_grouped_dot_comma":
		normalized, currency, err = normalizeAmountByProfile(v, format)
	default:
		return 0, fmt.Errorf("unsupported amount_format")
	}
	if err != nil {
		return 0, err
	}
	if currency != "" && currency != "EUR" {
		return 0, fmt.Errorf("unsupported currency: %s (only EUR is allowed)", currency)
	}
	if !reAmount.MatchString(normalized) {
		return 0, fmt.Errorf("invalid amount")
	}

	normalized = strings.ReplaceAll(normalized, ",", ".")
	parts := strings.SplitN(normalized, ".", 2)

	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}

	if len(frac) == 1 {
		frac += "0"
	} else if len(frac) == 0 {
		frac = "00"
	}

	var cents int64
	for _, r := range whole {
		cents = cents*10 + int64(r-'0')
	}
	cents *= 100
	if frac != "" {
		cents += int64(frac[0]-'0')*10 + int64(frac[1]-'0')
	}
	return cents, nil
}

func normalizeAmountByProfile(v, format string) (string, string, error) {
	normalized, currency, err := normalizeAmountInput(v)
	if err != nil {
		return "", "", err
	}
	normalized = strings.TrimSpace(normalized)
	switch format {
	case "eur_dot":
		if !reAmountEURDot.MatchString(normalized) {
			return "", "", fmt.Errorf("invalid amount")
		}
		return normalized, currency, nil
	case "eur_comma":
		if !reAmountEURComma.MatchString(normalized) {
			return "", "", fmt.Errorf("invalid amount")
		}
		return normalized, currency, nil
	case "eur_grouped_space_comma":
		if strings.Contains(normalized, ".") {
			return "", "", fmt.Errorf("invalid amount")
		}
		if strings.Contains(normalized, " ") {
			if !reAmountEURGroupedSpaceComma.MatchString(normalized) {
				return "", "", fmt.Errorf("invalid amount")
			}
		} else if !reAmountEURComma.MatchString(normalized) {
			return "", "", fmt.Errorf("invalid amount")
		}
		return strings.ReplaceAll(normalized, " ", ""), currency, nil
	case "eur_grouped_dot_comma":
		if strings.Contains(normalized, ".") {
			if !reAmountEURGroupedDotComma.MatchString(normalized) {
				return "", "", fmt.Errorf("invalid amount")
			}
		} else if !reAmountEURComma.MatchString(normalized) {
			return "", "", fmt.Errorf("invalid amount")
		}
		return strings.ReplaceAll(normalized, ".", ""), currency, nil
	default:
		return "", "", fmt.Errorf("unsupported amount_format")
	}
}

func normalizeAmountInput(v string) (string, string, error) {
	upper := strings.ToUpper(v)
	hasEUR := strings.Contains(upper, "EUR") || strings.Contains(upper, "EURO") || strings.Contains(v, "€")
	hasUSD := strings.Contains(upper, "USD") || strings.Contains(v, "$")
	if hasEUR && hasUSD {
		return "", "", fmt.Errorf("invalid amount")
	}

	currency := ""
	if hasEUR {
		currency = "EUR"
	}
	if hasUSD {
		currency = "USD"
	}

	normalized := upper
	normalized = strings.ReplaceAll(normalized, "EURO", "")
	normalized = strings.ReplaceAll(normalized, "EUR", "")
	normalized = strings.ReplaceAll(normalized, "USD", "")
	normalized = strings.ReplaceAll(normalized, "€", "")
	normalized = strings.ReplaceAll(normalized, "$", "")
	normalized = strings.Join(strings.Fields(normalized), " ")

	if normalized == "" {
		return "", "", fmt.Errorf("invalid amount")
	}
	return normalized, currency, nil
}

func normalizeAmountInputLenient(v string) (string, string, error) {
	mapped := mapOCRDigits(v)
	currency := detectCurrency(mapped)
	if currency != "" && currency != "EUR" {
		return "", currency, nil
	}

	// Keep only digits and separators for amount reconstruction.
	var b strings.Builder
	for _, r := range mapped {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			b.WriteRune(r)
		}
	}
	numeric := b.String()
	if numeric == "" {
		return "", "", fmt.Errorf("invalid amount")
	}

	normalized, err := normalizeNumericSeparators(numeric)
	if err != nil {
		return "", "", err
	}
	return normalized, currency, nil
}

func mapOCRDigits(v string) string {
	rs := []rune(v)
	out := make([]rune, 0, len(rs))
	for i, r := range rs {
		switch r {
		case 'O', 'o':
			if hasNumericNeighbor(rs, i) {
				out = append(out, '0')
				continue
			}
		case 'I', 'l':
			if hasNumericNeighbor(rs, i) {
				out = append(out, '1')
				continue
			}
		}
		out = append(out, r)
	}
	return string(out)
}

func hasNumericNeighbor(rs []rune, i int) bool {
	for _, j := range []int{i - 1, i + 1} {
		if j < 0 || j >= len(rs) {
			continue
		}
		r := rs[j]
		if (r >= '0' && r <= '9') || r == '.' || r == ',' || unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func detectCurrency(v string) string {
	upper := strings.ToUpper(v)
	has := func(s string) bool { return strings.Contains(upper, s) }

	if strings.Contains(v, "€") || has("EUR") || has("EURO") {
		if strings.Contains(v, "$") || has("USD") || has("US$") {
			return "USD"
		}
		return "EUR"
	}
	if strings.Contains(v, "$") || has("USD") || has("US$") {
		return "USD"
	}
	if strings.Contains(v, "£") || has("GBP") {
		return "GBP"
	}
	if strings.Contains(v, "¥") || has("JPY") {
		return "JPY"
	}
	if has("CHF") {
		return "CHF"
	}
	return ""
}

func normalizeNumericSeparators(v string) (string, error) {
	if strings.Count(v, ".")+strings.Count(v, ",") == 0 {
		return v, nil
	}

	lastDot := strings.LastIndex(v, ".")
	lastComma := strings.LastIndex(v, ",")

	decSep := rune(0)
	decPos := -1
	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastDot > lastComma {
			decSep, decPos = '.', lastDot
		} else {
			decSep, decPos = ',', lastComma
		}
	case lastDot >= 0:
		if len(v)-lastDot-1 <= 2 {
			decSep, decPos = '.', lastDot
		}
	case lastComma >= 0:
		if len(v)-lastComma-1 <= 2 {
			decSep, decPos = ',', lastComma
		}
	}

	var out strings.Builder
	for i, r := range v {
		if r >= '0' && r <= '9' {
			out.WriteRune(r)
			continue
		}
		if (r == '.' || r == ',') && i == decPos && r == decSep {
			out.WriteRune('.')
		}
	}

	n := out.String()
	if n == "" || n == "." {
		return "", fmt.Errorf("invalid amount")
	}
	if strings.HasPrefix(n, ".") {
		n = "0" + n
	}
	if strings.HasSuffix(n, ".") {
		n = strings.TrimSuffix(n, ".")
	}
	if strings.Count(n, ".") > 1 {
		return "", fmt.Errorf("invalid amount")
	}

	parts := strings.SplitN(n, ".", 2)
	if len(parts[0]) == 0 {
		return "", fmt.Errorf("invalid amount")
	}
	if len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > 2) {
		return "", fmt.Errorf("invalid amount")
	}
	if slices.Contains([]string{"", "."}, n) {
		return "", fmt.Errorf("invalid amount")
	}
	return n, nil
}
