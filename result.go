package skater

func GetResults[T any](results []any) []T {
	var filtered []T
	for _, result := range results {
		if value, ok := result.(T); ok {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
