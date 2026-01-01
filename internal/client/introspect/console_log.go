package introspect

import (
	"fmt"
)

// ConsoleLog prints a formatted log line for HTTP requests
func ConsoleLog(method, path string, status int) {
	statusColor := getStatusColor(status)
	methodColor := getMethodColor(method)

	fmt.Printf("%s%-7s%s %s%-4d%s %s\n",
		methodColor, method, colorReset,
		statusColor, status, colorReset,
		path,
	)
}

func getStatusColor(status int) string {
	switch {
	case status >= 200 && status < 300:
		return colorGreen
	case status >= 300 && status < 400:
		return colorCyan
	case status >= 400 && status < 500:
		return colorYellow
	case status >= 500:
		return colorRed
	default:
		return colorReset
	}
}

func getMethodColor(method string) string {
	switch method {
	case "GET":
		return colorBlue
	case "POST":
		return colorGreen
	case "PUT":
		return colorYellow
	case "DELETE":
		return colorRed
	case "PATCH":
		return colorMagenta
	default:
		return colorReset
	}
}

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
)
