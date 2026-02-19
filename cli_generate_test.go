package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	runErr := fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stdout: %v", err)
	}
	_ = r.Close()

	return buf.String(), runErr
}

func TestRunGenerateBatch_JSONOutput(t *testing.T) {
	in := `[{"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}]`
	inputPath := filepath.Join(t.TempDir(), "in.json")
	if err := os.WriteFile(inputPath, []byte(in), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runGenerate([]string{
			"--input", inputPath,
			"--format", "json",
		})
	})
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}

	var got struct {
		OK        bool `json:"ok"`
		Total     int  `json:"total"`
		Succeeded int  `json:"succeeded"`
		Failed    int  `json:"failed"`
		Items     []struct {
			OK          bool   `json:"ok"`
			Payload     string `json:"payload"`
			AmountCents int64  `json:"amount_cents"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nout=%q", err, out)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items len=%d, want 1", len(got.Items))
	}
	if !got.OK || got.Total != 1 || got.Succeeded != 1 || got.Failed != 0 {
		t.Fatalf("unexpected summary: ok=%v total=%d succeeded=%d failed=%d", got.OK, got.Total, got.Succeeded, got.Failed)
	}
	if !got.Items[0].OK {
		t.Fatalf("item should be ok")
	}
	if got.Items[0].AmountCents != 4990 {
		t.Fatalf("amount_cents=%d, want 4990", got.Items[0].AmountCents)
	}
	if !strings.Contains(got.Items[0].Payload, "EUR49.90") {
		t.Fatalf("payload missing amount: %q", got.Items[0].Payload)
	}
}

func TestRunGenerateBatch_PartialFailuresReturnError(t *testing.T) {
	in := `[
{"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"},
{"name":"","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}
]`
	inputPath := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(inputPath, []byte(in), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runGenerateBatch(inputPath, "-", "json")
	})
	if err == nil {
		t.Fatalf("expected error for partial failures")
	}
	if !strings.Contains(err.Error(), "failed item") {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct {
		OK        bool `json:"ok"`
		Total     int  `json:"total"`
		Succeeded int  `json:"succeeded"`
		Failed    int  `json:"failed"`
		Items     []struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		} `json:"items"`
	}
	if uErr := json.Unmarshal([]byte(out), &got); uErr != nil {
		t.Fatalf("unmarshal output: %v\nout=%q", uErr, out)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items len=%d, want 2", len(got.Items))
	}
	if got.OK || got.Total != 2 || got.Succeeded != 1 || got.Failed != 1 {
		t.Fatalf("unexpected summary: ok=%v total=%d succeeded=%d failed=%d", got.OK, got.Total, got.Succeeded, got.Failed)
	}
	if !got.Items[0].OK {
		t.Fatalf("first item should be ok")
	}
	if got.Items[1].OK || got.Items[1].Error == "" {
		t.Fatalf("second item should be failed with error")
	}
}

func TestRunGenerateBatch_PNGStdoutNotAllowed(t *testing.T) {
	in := `[{"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}]`
	inputPath := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(inputPath, []byte(in), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	err := runGenerateBatch(inputPath, "-", "png")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "does not support --out -") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGenerateBatch_PNGOutputHasSummary(t *testing.T) {
	in := `[{"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}]`
	inputPath := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(inputPath, []byte(in), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "out")
	out, err := captureStdout(t, func() error {
		return runGenerateBatch(inputPath, outDir, "png")
	})
	if err != nil {
		t.Fatalf("runGenerateBatch: %v", err)
	}

	var got struct {
		OK        bool `json:"ok"`
		Total     int  `json:"total"`
		Succeeded int  `json:"succeeded"`
		Failed    int  `json:"failed"`
		Items     []struct {
			OK      bool   `json:"ok"`
			OutFile string `json:"out_file"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal output: %v\nout=%q", err, out)
	}
	if !got.OK || got.Total != 1 || got.Succeeded != 1 || got.Failed != 0 {
		t.Fatalf("unexpected summary: ok=%v total=%d succeeded=%d failed=%d", got.OK, got.Total, got.Succeeded, got.Failed)
	}
	if len(got.Items) != 1 || !got.Items[0].OK || got.Items[0].OutFile == "" {
		t.Fatalf("unexpected items: %#v", got.Items)
	}
	if _, err := os.Stat(got.Items[0].OutFile); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestRunGenerate_UnsupportedScheme(t *testing.T) {
	err := runGenerate([]string{
		"--scheme", "pix",
		"--name", "Example GmbH",
		"--iban", "DE12500105170648489890",
		"--bic", "INGDDEFFXXX",
		"--amount", "49.90",
		"--out", "x.png",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unsupported scheme") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGenerate_AmountFormat(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runGenerate([]string{
			"--name", "Example GmbH",
			"--iban", "DE12500105170648489890",
			"--bic", "INGDDEFFXXX",
			"--amount", "49,90",
			"--amount-format", "eur_comma",
			"--format", "payload",
		})
	})
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}
	if !strings.Contains(out, "EUR49.90") {
		t.Fatalf("payload missing normalized amount: %q", out)
	}
}
