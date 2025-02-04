package utils

func BytesToKB(bytes int64) float64 {
	return float64(bytes) / 1024
}

func BytesToMB(bytes int64) float64 {
	return float64(bytes) / (1024 * 1024)
}

func BytesToGB(bytes int64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}
