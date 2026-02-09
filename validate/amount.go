package validate

import (
	"fmt"
	"regexp"
	"strings"
)

var reAmount = regexp.MustCompile(`^\d{1,12}([.,]\d{1,2})?$`)

func parseAmountEUR(s string) (int64, error) {
	v := strings.TrimSpace(s)
	if v == "" {
		return 0, fmt.Errorf("amount is required")
	}
	if !reAmount.MatchString(v) {
		return 0, fmt.Errorf("invalid amount")
	}

	v = strings.ReplaceAll(v, ",", ".")
	parts := strings.SplitN(v, ".", 2)

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
