package domain

type MissingValueReason string

const (
	MissingValueReasonNotAcquired   MissingValueReason = "not_acquired"
	MissingValueReasonNotAnnounced  MissingValueReason = "not_announced"
	MissingValueReasonNotApplicable MissingValueReason = "not_applicable"
	MissingValueReasonUnavailable   MissingValueReason = "unavailable"
)
