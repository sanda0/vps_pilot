package utils

func BytesToKB(bytes float64) float64 {
	return float64(bytes) / 1024
}

func BytesToMB(bytes float64) float64 {
	return float64(bytes) / (1024 * 1024)
}

func BytesToGB(bytes float64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}
