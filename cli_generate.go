package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/safe-cap/sepaqx/qr"
	"github.com/safe-cap/sepaqx/validate"
)

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	name := fs.String("name", "", "receiver name")
	scheme := fs.String("scheme", "", "QR scheme (default: epc_sct)")
	iban := fs.String("iban", "", "receiver IBAN")
	bic := fs.String("bic", "", "receiver BIC")
	amount := fs.String("amount", "", "amount in EUR (example: 49.90)")
	amountFormat := fs.String("amount-format", "", "amount format profile (optional): eur_dot|eur_comma|eur_grouped_space_comma|eur_grouped_dot_comma|auto_eur_lenient")
	purpose := fs.String("purpose", "", "purpose code (defaults to GDDS)")
	remRef := fs.String("remittance-reference", "", "structured remittance reference")
	remText := fs.String("remittance-text", "", "unstructured remittance text")
	info := fs.String("information", "", "additional information")
	input := fs.String("input", "", "path to JSON array with batch input records")
	out := fs.String("out", "sepa-qr.png", "output file path (single) or output directory (batch), or - for stdout")
	format := fs.String("format", "png", "output format: png|payload|json")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	if strings.TrimSpace(*input) != "" {
		return runGenerateBatch(*input, *out, strings.ToLower(strings.TrimSpace(*format)))
	}

	in := validate.Input{
		Scheme:              *scheme,
		Name:                *name,
		IBAN:                *iban,
		BIC:                 *bic,
		Amount:              *amount,
		AmountFormat:        *amountFormat,
		Purpose:             *purpose,
		RemittanceReference: *remRef,
		RemittanceText:      *remText,
		Information:         *info,
	}
	return runGenerateOne(in, *out, strings.ToLower(strings.TrimSpace(*format)))
}

func runGenerateOne(in validate.Input, out, format string) error {
	cleaned, payload, err := buildPayload(in)
	if err != nil {
		return err
	}

	switch format {
	case "payload":
		_, err = fmt.Fprintln(os.Stdout, payload)
		return err
	case "json":
		resp := map[string]any{
			"ok":           true,
			"payload":      payload,
			"amount_cents": cleaned.AmountCents,
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	case "png":
		pngBytes, err := qr.MakeQR(payload, qr.DefaultPublicOptions())
		if err != nil {
			return err
		}
		recolored, err := qr.Recolor(pngBytes, "#000000", "transparent")
		if err == nil {
			pngBytes = recolored
		}
		return writeOutput(out, pngBytes)
	default:
		return errors.New("invalid --format, use: png|payload|json")
	}
}

func runGenerateBatch(inputPath, out, format string) error {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	inputs, err := parseBatchInputs(raw)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errors.New("input batch is empty")
	}

	type batchItem struct {
		Index       int    `json:"index"`
		OK          bool   `json:"ok"`
		Payload     string `json:"payload,omitempty"`
		AmountCents int64  `json:"amount_cents,omitempty"`
		Error       string `json:"error,omitempty"`
		OutFile     string `json:"out_file,omitempty"`
	}
	items := make([]batchItem, 0, len(inputs))
	failures := 0

	if format == "png" {
		if strings.TrimSpace(out) == "-" {
			return errors.New("batch png mode does not support --out -")
		}
		if err := os.MkdirAll(out, 0o755); err != nil {
			return err
		}
	}

	for i, in := range inputs {
		cleaned, payload, err := buildPayload(in)
		if err != nil {
			items = append(items, batchItem{Index: i, OK: false, Error: err.Error()})
			failures++
			continue
		}

		item := batchItem{
			Index:       i,
			OK:          true,
			Payload:     payload,
			AmountCents: cleaned.AmountCents,
		}

		if format == "png" {
			pngBytes, err := qr.MakeQR(payload, qr.DefaultPublicOptions())
			if err != nil {
				items = append(items, batchItem{Index: i, OK: false, Error: err.Error()})
				failures++
				continue
			}
			recolored, err := qr.Recolor(pngBytes, "#000000", "transparent")
			if err == nil {
				pngBytes = recolored
			}
			filePath := filepath.Join(out, fmt.Sprintf("sepa-qr-%d.png", i+1))
			if err := writeOutput(filePath, pngBytes); err != nil {
				items = append(items, batchItem{Index: i, OK: false, Error: err.Error()})
				failures++
				continue
			}
			item.OutFile = filePath
		}
		items = append(items, item)
	}

	switch format {
	case "png", "json":
		succeeded := len(items) - failures
		if succeeded < 0 {
			succeeded = 0
		}
		resp := map[string]any{
			"ok":        failures == 0,
			"total":     len(items),
			"succeeded": succeeded,
			"failed":    failures,
			"items":     items,
		}
		if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
			return err
		}
	case "payload":
		for _, item := range items {
			if !item.OK {
				_, _ = fmt.Fprintf(os.Stdout, "#%d error: %s\n", item.Index, item.Error)
				continue
			}
			_, _ = fmt.Fprintf(os.Stdout, "#%d\n%s\n", item.Index, item.Payload)
		}
	default:
		return errors.New("invalid --format, use: png|payload|json")
	}

	if failures > 0 {
		return fmt.Errorf("batch completed with %d failed item(s)", failures)
	}
	return nil
}

func parseBatchInputs(raw []byte) ([]validate.Input, error) {
	var asArray []validate.Input
	if err := json.Unmarshal(raw, &asArray); err == nil {
		return asArray, nil
	}

	var wrapped struct {
		Items []validate.Input `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		if wrapped.Items == nil {
			return nil, errors.New("invalid --input JSON: object must contain items array")
		}
		return wrapped.Items, nil
	}

	return nil, errors.New("invalid --input JSON: expected [] or {\"items\":[]}")
}

func buildPayload(in validate.Input) (*validate.Clean, string, error) {
	cleaned, err := validate.CleanAndValidate(in)
	if err != nil {
		return nil, "", err
	}
	if cleaned.Scheme != "epc_sct" {
		return nil, "", errors.New("unsupported scheme")
	}
	payload, err := qr.BuildEPCPayload(
		cleaned.Name,
		cleaned.IBAN,
		cleaned.BIC,
		cleaned.AmountCents,
		cleaned.Purpose,
		cleaned.RemittanceReference,
		cleaned.RemittanceText,
		cleaned.Information,
	)
	if err != nil {
		return nil, "", err
	}
	return cleaned, payload, nil
}

func writeOutput(path string, data []byte) error {
	if strings.TrimSpace(path) == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
