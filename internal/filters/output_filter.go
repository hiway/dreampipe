package filters

// OutputFilter defines the interface for an output filter.
// Filters are applied to the LLM output before it is written to stdout.
type OutputFilter interface {
	Apply(input string) string
}
