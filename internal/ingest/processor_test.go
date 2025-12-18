package ingest

import (
	"errors"
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
)

func validProductForProcessor(key string) domain.Product {
	p := domain.Product{
		ProductKey:   key,
		Title:        "Test",
		Description:  "Desc",
		Link:         "https://example.com/p/" + key,
		ImageLink:    "https://example.com/p/" + key + ".jpg",
		Condition:    "new",
		Availability: "in_stock",
		Price: domain.Money{
			AmountDecimal: "19.99",
			Currency:      "USD",
		},
		Channel: domain.ChannelFields{
			Google: &domain.GoogleFields{
				Control: domain.ChannelControl{State: domain.ChannelStateActive},
			},
		},
	}
	return p
}

func TestProcessor_RejectsInvalidBase(t *testing.T) {
	proc := NewProcessor()

	p := validProductForProcessor("sku1")
	p.Title = ""

	out, err := proc.ProcessProducts([]domain.Product{p}, []string{"google"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Summary.Received != 1 || out.Summary.Rejected != 1 {
		t.Fatalf("unexpected summary: %#v", out.Summary)
	}

	if out.Products[0].Disposition != domain.ProductDispositionRejected {
		t.Fatalf("expected rejected, got %s", out.Products[0].Disposition)
	}
	if out.Products[0].Reason != "base_validation_failed" {
		t.Fatalf("expected base_validation_failed, got %s", out.Products[0].Reason)
	}
	if len(out.Products[0].Issues) == 0 {
		t.Fatalf("expected issues")
	}
}

func TestProcessor_RejectsMissingChannelBlockWhenEnabled(t *testing.T) {
	proc := NewProcessor()

	p := validProductForProcessor("sku1")
	p.Channel.Google = nil

	out, err := proc.ProcessProducts([]domain.Product{p}, []string{"google"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Summary.Rejected != 1 {
		t.Fatalf("expected rejected=1, got %#v", out.Summary)
	}

	if out.Products[0].Reason != "channel_validation_failed" {
		t.Fatalf("expected channel_validation_failed, got %s", out.Products[0].Reason)
	}
}

func TestProcessor_EnqueuesNewWhenNoPreviousHash(t *testing.T) {
	proc := NewProcessor()

	p := validProductForProcessor("sku1")

	out, err := proc.ProcessProducts([]domain.Product{p}, []string{"google"}, func(productKey string) (string, bool, error) {
		return "", false, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Summary.Enqueued != 1 || out.Summary.Unchanged != 0 || out.Summary.Rejected != 0 {
		t.Fatalf("unexpected summary: %#v", out.Summary)
	}

	if out.Products[0].Disposition != domain.ProductDispositionEnqueued {
		t.Fatalf("expected enqueued, got %s", out.Products[0].Disposition)
	}
	if out.Products[0].Reason != "new_product" {
		t.Fatalf("expected new_product, got %s", out.Products[0].Reason)
	}
	if out.Products[0].Hash == "" {
		t.Fatalf("expected hash to be set")
	}
}

func TestProcessor_UnchangedWhenSameHash(t *testing.T) {
	proc := NewProcessor()

	p := validProductForProcessor("sku1")
	hash, err := proc.Hasher.HashNormalized(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, err := proc.ProcessProducts([]domain.Product{p}, []string{"google"}, func(productKey string) (string, bool, error) {
		return hash, true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Summary.Unchanged != 1 || out.Summary.Enqueued != 0 || out.Summary.Rejected != 0 {
		t.Fatalf("unexpected summary: %#v", out.Summary)
	}

	if out.Products[0].Disposition != domain.ProductDispositionUnchanged {
		t.Fatalf("expected unchanged, got %s", out.Products[0].Disposition)
	}
	if out.Products[0].Reason != "no_change_detected" {
		t.Fatalf("expected no_change_detected, got %s", out.Products[0].Reason)
	}
}

func TestProcessor_PropagatesLookupError(t *testing.T) {
	proc := NewProcessor()

	p := validProductForProcessor("sku1")
	wantErr := errors.New("lookup failed")

	_, err := proc.ProcessProducts([]domain.Product{p}, []string{"google"}, func(productKey string) (string, bool, error) {
		return "", false, wantErr
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("expected %q, got %q", wantErr.Error(), err.Error())
	}
}
