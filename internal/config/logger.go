package config

import "fmt"

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

func LogInfo(prefix, message string) {
	fmt.Printf("%s[INFO] [%s]%s %s\n", ColorCyan, prefix, ColorReset, message)
}

func LogSuccess(prefix, message string) {
	fmt.Printf("%s[OK]   [%s]%s %s\n", ColorGreen, prefix, ColorReset, message)
}

func LogError(prefix, message string) {
	fmt.Printf("%s[ERR]  [%s]%s %s\n", ColorRed, prefix, ColorReset, message)
}
