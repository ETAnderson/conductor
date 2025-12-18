package ingest

import (
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
)

func TestComputeDisposition_NewProduct(t *testing.T) {
	d := ComputeDisposition("", "abc")

	if d.Disposition != domain.ProductDispositionEnqueued {
		t.Fatalf("expected enqueued, got %s", d.Disposition)
	}
	if d.Reason != "new_product" {
		t.Fatalf("expected new_product, got %s", d.Reason)
	}
}

func TestComputeDisposition_Unchanged(t *testing.T) {
	d := ComputeDisposition("abc", "abc")

	if d.Disposition != domain.ProductDispositionUnchanged {
		t.Fatalf("expected unchanged, got %s", d.Disposition)
	}
	if d.Reason != "no_change_detected" {
		t.Fatalf("expected no_change_detected, got %s", d.Reason)
	}
}

func TestComputeDisposition_Changed(t *testing.T) {
	d := ComputeDisposition("abc", "def")

	if d.Disposition != domain.ProductDispositionEnqueued {
		t.Fatalf("expected enqueued, got %s", d.Disposition)
	}
	if d.Reason != "content_changed" {
		t.Fatalf("expected content_changed, got %s", d.Reason)
	}
}
