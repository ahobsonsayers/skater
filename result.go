package skater

func GetResults[T any](results []any) []T {
	var filtered []T
	for _, r := range results {
		if v, ok := r.(T); ok {
			filtered = append(filtered, v)
		}
	}
	return filtered
}