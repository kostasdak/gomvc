package gomvc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

var infoLog *log.Logger
var errorLog *log.Logger
var cfg *AppConfig

// InitHelpers is the function to call in order to build the Helpers
func InitHelpers(appcfg *AppConfig) {
	cfg = appcfg
	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime)
}

// ServerError print/log a Server error -> send to error logger
func ServerError(w http.ResponseWriter, err error) {
	var text string
	if cfg.ShowStackOnError {
		text = fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	} else {
		text = fmt.Sprintf("%s\n", err.Error())
	}

	errorLog.Println(text)
	if w != nil {
		http.Error(w, text, http.StatusInternalServerError)
	}
}

// InfoMessage print/log an INFO message -> send to info logger
func InfoMessage(info string) {
	if cfg.EnableInfoLog {
		infoLog.Println(info)
	}
}

// FindInSlice find a value in a slice and return the index
func FindInSlice(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}

// Helper function to get client IP
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (if behind proxy)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// authenticateLinuxUser validates against Linux password
func authenticateLinuxUser(username, password string) bool {
	// Validate username format (prevent injection)
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(username) {
		InfoMessage("Invalid username format: " + username)
		return false
	}

	if len(username) > 32 {
		InfoMessage("Username too long: " + username)
		return false
	}

	if len(password) == 0 {
		InfoMessage("Empty password")
		return false
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use Python to verify password against /etc/shadow
	// Escape single quotes in password for Python
	escapedPassword := strings.ReplaceAll(password, `\`, `\\`)
	escapedPassword = strings.ReplaceAll(escapedPassword, `'`, `\'`)

	pythonScript := fmt.Sprintf(`
import crypt
import sys

username = '%s'
password = '%s'

try:
    with open('/etc/shadow', 'r') as f:
        for line in f:
            parts = line.strip().split(':')
            if len(parts) >= 2 and parts[0] == username:
                stored_hash = parts[1]
                
                # Check if account is disabled
                if stored_hash in ['', '!', '*', '!!']:
                    sys.exit(1)
                
                # Verify password using crypt
                if crypt.crypt(password, stored_hash) == stored_hash:
                    sys.exit(0)
                else:
                    sys.exit(1)
    
    sys.exit(1)
    
except PermissionError:
    sys.exit(2)
except Exception:
    sys.exit(3)
`, username, escapedPassword)

	cmd := exec.CommandContext(ctx, "python3", "-c", pythonScript)
	err := cmd.Run()

	return err == nil
}

// CenterText centers a string within a specified width and surrounds it with a decorator character
// Examples:
//
//	CenterText("Hello", 20, '=')  → "====== Hello ======="
//	CenterText("GoMVC", 30, '-')  → "------------ GoMVC ------------"
//	CenterText("Title", 15, '*')  → "**** Title *****"
func CenterText(text string, length int, deco rune) string {
	// Add spaces before and after text
	textWithSpaces := " " + text + " "

	// Get actual text length (handles Unicode properly)
	textLen := len([]rune(textWithSpaces))

	// If text is longer than desired length, return as-is
	if textLen >= length {
		return textWithSpaces
	}

	// Calculate total padding needed
	totalPadding := length - textLen

	// Split padding between left and right
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	// Build the result
	var result strings.Builder
	result.Grow(length) // Pre-allocate space for efficiency

	// Add left padding
	for i := 0; i < leftPadding; i++ {
		result.WriteRune(deco)
	}

	// Add text with spaces
	result.WriteString(textWithSpaces)

	// Add right padding
	for i := 0; i < rightPadding; i++ {
		result.WriteRune(deco)
	}

	return result.String()
}

// CenterTextSpaced centers text with spaces and adds decorators at edges
// Examples:
//
//	CenterTextSpaced("Hello", 20, '|')  → "|      Hello       |"
//	CenterTextSpaced("GoMVC", 30, '*')  → "*          GoMVC           *"
func CenterTextSpaced(text string, length int, deco rune) string {
	textLen := len([]rune(text))

	// Need at least 2 characters for decorators
	if length < 2 {
		return text
	}

	// If text is too long for decorators, return text only
	if textLen >= length-2 {
		return text
	}

	// Available space for text and padding (minus 2 decorators)
	availableSpace := length - 2
	totalPadding := availableSpace - textLen
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	var result strings.Builder
	result.Grow(length)

	// Left decorator
	result.WriteRune(deco)

	// Left padding
	for i := 0; i < leftPadding; i++ {
		result.WriteRune(' ')
	}

	// Text
	result.WriteString(text)

	// Right padding
	for i := 0; i < rightPadding; i++ {
		result.WriteRune(' ')
	}

	// Right decorator
	result.WriteRune(deco)

	return result.String()
}

// CreateBanner creates a multi-line banner with title and subtitle
// Example:
//
//	banner := CreateBanner("GoMVC Server", "Port 8080", 40, '=')
//	for _, line := range banner {
//	    fmt.Println(line)
//	}
func CreateBanner(title, subtitle string, width int, deco rune) []string {
	var lines []string

	// Top border
	lines = append(lines, strings.Repeat(string(deco), width))

	// Empty line
	lines = append(lines, CenterTextSpaced("", width, deco))

	// Title
	lines = append(lines, CenterTextSpaced(title, width, deco))

	// Subtitle (if provided)
	if subtitle != "" {
		lines = append(lines, CenterTextSpaced(subtitle, width, deco))
	}

	// Empty line
	lines = append(lines, CenterTextSpaced("", width, deco))

	// Bottom border
	lines = append(lines, strings.Repeat(string(deco), width))

	return lines
}

// CreateBox creates a boxed text with decorators
// Example:
//
//	lines := CreateBox("Hello", 15, '=')
//	for _, line := range lines {
//	    fmt.Println(line)
//	}
//	Output:
//	===============
//	=====Hello=====
//	===============
func CreateBox(text string, width int, deco rune) []string {
	lines := make([]string, 3)

	// Top line
	lines[0] = strings.Repeat(string(deco), width)

	// Middle line with text
	lines[1] = CenterText(text, width, deco)

	// Bottom line
	lines[2] = strings.Repeat(string(deco), width)

	return lines
}
