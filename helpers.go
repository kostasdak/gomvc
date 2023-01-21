package gomvc

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
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
