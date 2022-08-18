package modifier

// My idea for now was to create some sort of modifiers
// that consume []byte input, modify it accordingly to the
// provided configuration, and return []byte that Creator can
// write to a file.
type Modifier interface {
	Execute([]byte) ([]byte, error)
}
