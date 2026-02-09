package qr

import (
	"strings"
	"testing"
)

func TestBuildEPCPayload_OrderAndEmptyLines(t *testing.T) {
	payload, err := BuildEPCPayload(
		"Example GmbH",
		"DE12500105170648489890",
		"INGDDEFFXXX",
		4990,
		"GDDS",
		"RF18539007547034",
		"",
		"Invoice 0001",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(payload, "\n")
	if len(lines) != 12 {
		t.Fatalf("expected 12 lines, got %d", len(lines))
	}
	if lines[0] != "BCD" || lines[1] != "001" || lines[2] != "1" || lines[3] != "SCT" {
		t.Fatalf("unexpected header lines: %v", lines[:4])
	}
	if lines[4] != "INGDDEFFXXX" || lines[5] != "Example GmbH" || lines[6] != "DE12500105170648489890" {
		t.Fatalf("unexpected creditor lines: %v", lines[4:7])
	}
	if lines[7] != "EUR49.90" {
		t.Fatalf("unexpected amount: %q", lines[7])
	}
	// Required order after amount: purpose -> remittanceRef -> remittanceText -> info
	if lines[8] != "GDDS" || lines[9] != "RF18539007547034" || lines[10] != "" || lines[11] != "Invoice 0001" {
		t.Fatalf("unexpected tail lines: %v", lines[8:12])
	}
}

func TestBuildEPCPayload_MutualExclusion(t *testing.T) {
	_, err := BuildEPCPayload(
		"Example GmbH",
		"DE12500105170648489890",
		"INGDDEFFXXX",
		100,
		"GDDS",
		"RF18539007547034",
		"Text",
		"",
	)
	if err == nil {
		t.Fatalf("expected error when remittance ref and text are both set")
	}
}
