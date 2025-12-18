package ingest

import (
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
)

func baseProductForHash() domain.Product {
	return domain.Product{
		ProductKey:   "sku1",
		GroupKey:     "group1",
		Title:        "Test",
		Description:  "Desc",
		Link:         "https://example.com/p/sku1",
		ImageLink:    "https://example.com/p/sku1.jpg",
		Condition:    "new",
		Availability: "in_stock",
		Price: domain.Money{
			AmountDecimal: "19.99",
			Currency:      "USD",
		},
		AdditionalImageLinks: []string{"https://example.com/a.jpg", "https://example.com/b.jpg"},
		Options: map[string]string{
			"size":  "M",
			"color": "red",
		},
		Attributes: map[string]any{
			"material": "cotton",
			"weight":   12,
		},
		Channel: domain.ChannelFields{
			Google: &domain.GoogleFields{
				Control: domain.ChannelControl{State: domain.ChannelStateActive},
			},
		},
	}
}

func TestHashNormalized_IsDeterministicAcrossMapOrder(t *testing.T) {
	h := Hasher{}

	p1 := baseProductForHash()
	p2 := baseProductForHash()

	// Rebuild maps with different insertion order
	p2.Options = map[string]string{
		"color": "red",
		"size":  "M",
	}
	p2.Attributes = map[string]any{
		"weight":   12,
		"material": "cotton",
	}

	hash1, err := h.HashNormalized(p1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash2, err := h.HashNormalized(p2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("expected same hash, got %s vs %s", hash1, hash2)
	}
}

func TestHashNormalized_SortsAdditionalImageLinks(t *testing.T) {
	h := Hasher{}

	p1 := baseProductForHash()
	p2 := baseProductForHash()

	p1.AdditionalImageLinks = []string{"https://example.com/b.jpg", "https://example.com/a.jpg"}
	p2.AdditionalImageLinks = []string{"https://example.com/a.jpg", "https://example.com/b.jpg"}

	hash1, err := h.HashNormalized(p1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash2, err := h.HashNormalized(p2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("expected same hash (order-insensitive), got %s vs %s", hash1, hash2)
	}
}

func TestHashNormalized_ChangesWhenLifecycleStateChanges(t *testing.T) {
	h := Hasher{}

	p1 := baseProductForHash()
	p2 := baseProductForHash()
	p2.Channel.Google.Control.State = domain.ChannelStateDelete

	hash1, err := h.HashNormalized(p1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash2, err := h.HashNormalized(p2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1 == hash2 {
		t.Fatalf("expected different hashes when control.state changes")
	}
}

func TestHashNormalized_ChangesWhenCoreFieldChanges(t *testing.T) {
	h := Hasher{}

	p1 := baseProductForHash()
	p2 := baseProductForHash()
	p2.Title = "Different Title"

	hash1, err := h.HashNormalized(p1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hash2, err := h.HashNormalized(p2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1 == hash2 {
		t.Fatalf("expected different hashes when title changes")
	}
}
