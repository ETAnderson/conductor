package ingest

import "github.com/ETAnderson/conductor/internal/domain"

type DeltaDecision struct {
	Disposition domain.ProductDisposition `json:"disposition"`
	Reason      string                    `json:"reason"`
}

func ComputeDisposition(previousHash string, currentHash string) DeltaDecision {
	// If we have no previous hash, it is a new product (needs push).
	if previousHash == "" {
		return DeltaDecision{
			Disposition: domain.ProductDispositionEnqueued,
			Reason:      "new_product",
		}
	}

	// If hashes match, nothing changed.
	if previousHash == currentHash {
		return DeltaDecision{
			Disposition: domain.ProductDispositionUnchanged,
			Reason:      "no_change_detected",
		}
	}

	// Otherwise it changed and must be pushed.
	return DeltaDecision{
		Disposition: domain.ProductDispositionEnqueued,
		Reason:      "content_changed",
	}
}
