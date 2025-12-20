package google

import (
	"context"
	"testing"

	"github.com/ETAnderson/conductor/internal/channels"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestGoogleBuild_MissingDoc(t *testing.T) {
	st := state.NewMemoryStore()
	ch := Channel{Store: st}

	res, err := ch.Build(context.Background(), 1, []channels.ProductRef{
		{ProductKey: "sku1", Hash: "h1"},
	})
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if res.ErrCount != 1 || res.OkCount != 0 {
		t.Fatalf("expected ok=0 err=1 got ok=%d err=%d", res.OkCount, res.ErrCount)
	}
	if len(res.Items) != 1 || res.Items[0].Message != "missing_product_doc" {
		t.Fatalf("unexpected item: %+v", res.Items)
	}
}

func TestGoogleBuild_RequiresDescription(t *testing.T) {
	st := state.NewMemoryStore()
	tenantID := uint64(1)

	_ = st.UpsertProductDoc(context.Background(), tenantID, "sku1", state.ProductDocRecord{
		ProductJSON: []byte(`{
			"product_key":"sku1",
			"title":"T",
			"description":"",
			"link":"https://example.com/p/sku1",
			"image_link":"https://example.com/p/sku1.jpg",
			"condition":"new",
			"availability":"in_stock",
			"price":{"amount_decimal":"19.99","currency":"USD"}
		}`),
	})

	ch := Channel{Store: st}

	res, err := ch.Build(context.Background(), tenantID, []channels.ProductRef{
		{ProductKey: "sku1", Hash: "h1"},
	})
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if res.ErrCount != 1 || res.OkCount != 0 {
		t.Fatalf("expected ok=0 err=1 got ok=%d err=%d", res.OkCount, res.ErrCount)
	}
	if res.Items[0].Message != "missing_description" {
		t.Fatalf("unexpected item: %+v", res.Items[0])
	}
}

func TestGoogleBuild_Ok(t *testing.T) {
	st := state.NewMemoryStore()
	tenantID := uint64(1)

	_ = st.UpsertProductDoc(context.Background(), tenantID, "sku1", state.ProductDocRecord{
		ProductJSON: []byte(`{
			"product_key":"sku1",
			"title":"T",
			"description":"D",
			"link":"https://example.com/p/sku1",
			"image_link":"https://example.com/p/sku1.jpg",
			"condition":"new",
			"availability":"in_stock",
			"price":{"amount_decimal":"19.99","currency":"USD"}
		}`),
	})

	ch := Channel{Store: st}

	res, err := ch.Build(context.Background(), tenantID, []channels.ProductRef{
		{ProductKey: "sku1", Hash: "h1"},
	})
	if err != nil {
		t.Fatalf("Build err: %v", err)
	}
	if res.OkCount != 1 || res.ErrCount != 0 {
		t.Fatalf("expected ok=1 err=0 got ok=%d err=%d", res.OkCount, res.ErrCount)
	}
	if res.Items[0].Status != "ok" || res.Items[0].Message != "google_item_built" {
		t.Fatalf("unexpected item: %+v", res.Items[0])
	}
}
