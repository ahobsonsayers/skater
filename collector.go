package skater

import (
	"reflect"
	"sync"
)

// ResultCollector collects results into bucket depending on their types.
// This allows you to access results as their full type, rather than as any or interface{}
type ResultCollector struct {
	mu      sync.Mutex
	buckets map[reflect.Type][]any
}

func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		buckets: make(map[reflect.Type][]any),
	}
}

// AddResults adds results to the collector in the correct bucket.
// It is safe for concurrent use.
func (rc *ResultCollector) AddResults(results ...any) {
	if len(results) == 0 {
		return
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	for _, result := range results {
		if results == nil {
			continue
		}
		t := reflect.TypeOf(result)
		rc.buckets[t] = append(rc.buckets[t], result)
	}
}

// GetResults gets all results of type T stored in the collector.
// Returns nil if no results of that type are stored.
func GetResults[T any](rc *ResultCollector) []T {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	resultType := reflect.TypeOf((*T)(nil)).Elem()
	values, ok := rc.buckets[resultType]
	if !ok {
		return nil
	}

	results := make([]T, 0, len(values))
	for _, value := range values {
		result := value.(T)
		results = append(results, result)
	}

	return results
}
