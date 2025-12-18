package domain

type ProductDisposition string

const (
	ProductDispositionRejected  ProductDisposition = "rejected"
	ProductDispositionUnchanged ProductDisposition = "unchanged"
	ProductDispositionEnqueued  ProductDisposition = "enqueued"
)
