package main

func mapElements[T any, V any](slice []T, transform func(T) V) []V {
	result := make([]V, 0, len(slice))
	for _, v := range slice {
		result = append(result, transform(v))
	}
	return result
}

type empty = struct{}
