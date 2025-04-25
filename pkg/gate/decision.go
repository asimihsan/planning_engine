package gate

// Decision represents the outcome of a successful policy evaluation.
type Decision struct {
	Allow       bool
	DenyReasons []string
}
