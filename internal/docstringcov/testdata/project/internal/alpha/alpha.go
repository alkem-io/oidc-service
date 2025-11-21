package alpha

// Greeter coordinates greeting workflows for analyzer fixtures.
type Greeter struct{}

// Greet produces a deterministic greeting used in tests.
func (Greeter) Greet() string {
	return "hello"
}
