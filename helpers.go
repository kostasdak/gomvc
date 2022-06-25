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

func InitHelpers(appcfg *AppConfig) {
	cfg = appcfg
	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime)
}

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

func InfoMessage(info string) {
	if cfg.EnableInfoLog {
		infoLog.Println(info)
	}
}

func FindInSlice(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}
