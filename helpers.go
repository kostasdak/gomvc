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

// authenticateLinuxUser validates against Linux using 'su' command
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

	// 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute 'su -c exit username'
	cmd := exec.CommandContext(ctx, "su", "-c", "exit", username)
	cmd.Stdin = strings.NewReader(password + "\n")
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	return err == nil
}
