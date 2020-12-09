package pluralize

// Pluralize takes an English word in singular form and a count and returns
// that word appropriately pluralized for the count.
func Pluralize(singularWord string, count int) string {
	if count == 0 {
		return singularWord + "s"
	}

	if count == 1 {
		return singularWord
	}

	if count > 1 {
		return singularWord + "s"
	}

	return singularWord + "s"
}
