package execute

import (
	"context"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/channels"
	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

type fakeChannel struct {
	name  string
	calls int
	got   []channels.ProductRef
}

func (f *fakeChannel) Name() string { return f.name }

func (f *fakeChannel) Build(ctx context.Context, tenantID uint64, products []channels.ProductRef) (channels.BuildResult, error) {
	f.calls++
	f.got = append([]channels.ProductRef(nil), products...)

	return channels.BuildResult{
		Channel: f.name,
		OkCount: len(products),
		Items:   nil,
	}, nil
}

func TestExecutor_Channels_CallsGoogleWithEnqueuedRefsOnly(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_chan_1"
	tenantID := uint64(1)

	_ = st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "processing",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC(),
	})

	_ = st.InsertRunProducts(context.Background(), runID, []ingest.ProductProcessResult{
		{ProductKey: "sku1", Hash: "h1", Disposition: domain.ProductDispositionEnqueued},
		{ProductKey: "sku2", Hash: "h2", Disposition: domain.ProductDispositionUnchanged},
		{ProductKey: "sku3", Hash: "h3", Disposition: domain.ProductDispositionEnqueued},
	})

	fg := &fakeChannel{name: "google"}
	reg := channels.NewRegistry(fg)

	ex := Executor{
		Store:           st,
		Registry:        reg,
		EnabledChannels: []string{"google"},
	}

	if err := ex.Execute(context.Background(), runID, tenantID); err != nil {
		t.Fatalf("Execute err: %v", err)
	}

	if fg.calls != 1 {
		t.Fatalf("expected google called once, got %d", fg.calls)
	}
	if len(fg.got) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(fg.got))
	}
	if fg.got[0].ProductKey != "sku1" || fg.got[1].ProductKey != "sku3" {
		t.Fatalf("unexpected refs: %+v", fg.got)
	}
}

func TestExecutor_Channels_SkipsUnknownChannelName(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_chan_2"
	tenantID := uint64(1)

	_ = st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "processing",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC(),
	})

	_ = st.InsertRunProducts(context.Background(), runID, []ingest.ProductProcessResult{
		{ProductKey: "sku1", Hash: "h1", Disposition: domain.ProductDispositionEnqueued},
	})

	ex := Executor{
		Store:           st,
		Registry:        channels.NewRegistry(), // empty
		EnabledChannels: []string{"google"},
	}

	if err := ex.Execute(context.Background(), runID, tenantID); err != nil {
		t.Fatalf("Execute err: %v", err)
	}
}
