package validate

func ValidIBAN(iban string) bool {
	// Basic sanitization
	if len(iban) < 15 || len(iban) > 34 {
		return false
	}
	for _, r := range iban {
		if !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}

	// Move first 4 chars to the end
	rearranged := iban[4:] + iban[:4]

	// Convert letters to numbers (A=10..Z=35), compute mod 97
	mod := 0
	for _, r := range rearranged {
		if r >= '0' && r <= '9' {
			mod = (mod*10 + int(r-'0')) % 97
			continue
		}
		if r >= 'A' && r <= 'Z' {
			val := int(r-'A') + 10
			// two digits
			mod = (mod*10 + (val / 10)) % 97
			mod = (mod*10 + (val % 10)) % 97
			continue
		}
		return false
	}

	return mod == 1
}
