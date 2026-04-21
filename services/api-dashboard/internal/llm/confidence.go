package llm

// ConfidenceStatus is the outcome of a confidence-check approach.
type ConfidenceStatus string

const (
	// ConfidenceStatusOK means the check passed with high confidence.
	ConfidenceStatusOK ConfidenceStatus = "OK"
	// ConfidenceStatusUnsure means the check raised doubts; manual review recommended.
	ConfidenceStatusUnsure ConfidenceStatus = "UNSURE"
	// ConfidenceStatusFail means the check failed; result should not be used.
	ConfidenceStatusFail ConfidenceStatus = "FAIL"
)

// ApproachName identifies a confidence-estimation method.
type ApproachName string

const (
	ApproachConstraint ApproachName = "constraint"
	ApproachSelfCheck  ApproachName = "self_check"
)
