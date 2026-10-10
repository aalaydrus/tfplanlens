package analyze

// Analysis is the single input to every renderer (SPEC section 6).
// Field types are placeholders until the analysis is implemented.
type Analysis struct {
	Summary  Summary
	Findings []Finding
	Drift    []DriftItem
	Groups   []Group
	Changes  []Change
}

// Summary holds plan-wide counts (SPEC 5.1).
type Summary struct{}

// Finding is a danger finding such as a delete or replace (SPEC 5.2).
type Finding struct{}

// DriftItem is a change made outside Terraform (SPEC 5.3).
type DriftItem struct{}

// Group is an account/provider grouping of changes (SPEC 5.5).
type Group struct{}

// Change is a single resource change.
type Change struct{}
