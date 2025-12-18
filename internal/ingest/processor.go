package ingest

import (
	"github.com/ETAnderson/conductor/internal/domain"
)

type PreviousHashLookup func(productKey string) (string, bool, error)

type ProductProcessResult struct {
	ProductKey string `json:"product_key"`
	Hash       string `json:"hash,omitempty"`

	Disposition domain.ProductDisposition `json:"disposition"`
	Reason      string                    `json:"reason,omitempty"`

	Issues []ValidationIssue `json:"issues,omitempty"`
}

type ProcessSummary struct {
	Received  int `json:"received"`
	Valid     int `json:"valid"`
	Rejected  int `json:"rejected"`
	Unchanged int `json:"unchanged"`
	Enqueued  int `json:"enqueued"`
}

type ProcessOutput struct {
	Summary  ProcessSummary         `json:"summary"`
	Products []ProductProcessResult `json:"products"`
}

type Processor struct {
	Hasher Hasher
}

func NewProcessor() Processor {
	return Processor{
		Hasher: Hasher{},
	}
}

func (p Processor) ProcessProduct(prod domain.Product, enabledChannels []string, lookup PreviousHashLookup) (ProductProcessResult, bool, error) {
	res := ProductProcessResult{
		ProductKey: prod.ProductKey,
	}

	// Base validation
	base := ValidateProductBase(prod)
	if !base.IsValid() {
		res.Disposition = domain.ProductDispositionRejected
		res.Reason = "base_validation_failed"
		res.Issues = append(res.Issues, base.Issues...)
		return res, false, nil
	}

	// Channel control validation (only for enabled channels)
	ch := ValidateChannelControls(prod, enabledChannels)
	if !ch.IsValid() {
		res.Disposition = domain.ProductDispositionRejected
		res.Reason = "channel_validation_failed"
		res.Issues = append(res.Issues, ch.Issues...)
		return res, false, nil
	}

	// Hash normalized
	hash, err := p.Hasher.HashNormalized(prod)
	if err != nil {
		return ProductProcessResult{}, false, err
	}
	res.Hash = hash

	// Lookup previous hash
	prev := ""
	if lookup != nil {
		prevHash, ok, err := lookup(prod.ProductKey)
		if err != nil {
			return ProductProcessResult{}, false, err
		}
		if ok {
			prev = prevHash
		}
	}

	decision := ComputeDisposition(prev, hash)
	res.Disposition = decision.Disposition
	res.Reason = decision.Reason

	// valid = true
	return res, true, nil
}

func (p Processor) ProcessProducts(products []domain.Product, enabledChannels []string, lookup PreviousHashLookup) (ProcessOutput, error) {
	out := ProcessOutput{
		Summary: ProcessSummary{
			Received: len(products),
		},
		Products: make([]ProductProcessResult, 0, len(products)),
	}

	for _, prod := range products {
		res, valid, err := p.ProcessProduct(prod, enabledChannels, lookup)
		if err != nil {
			return ProcessOutput{}, err
		}

		out.Products = append(out.Products, res)

		if !valid {
			out.Summary.Rejected++
			continue
		}

		out.Summary.Valid++

		switch res.Disposition {
		case domain.ProductDispositionUnchanged:
			out.Summary.Unchanged++
		case domain.ProductDispositionEnqueued:
			out.Summary.Enqueued++
		}
	}

	return out, nil
}
