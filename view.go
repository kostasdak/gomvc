package gomvc

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
)

// TemplateData is used to provide all data to the template engine to build the webpage.
type TemplateData struct {
	Auth         AuthObject
	AuthExpired  bool
	Model        Model
	Result       []ResultRow
	URLParams    map[string][]interface{}
	CustomValues map[string][]interface{}
	CSRFToken    string
	Flash        string
	Warning      string
	Error        string
}

// ====================================================================== Template ready functions ======================================================================
// FindValue searches for a key in the section data and returns its value
func FindValue(data []interface{}, searchKey string) string {
	for _, item := range data {
		if str, ok := item.(string); ok {
			// For ParseSystemInfo format: "Hostname: kostas-server"
			if strings.Contains(str, ":") {
				parts := strings.SplitN(str, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					if key == searchKey {
						return value
					}
				}
			}
		} else if kvPair, ok := item.(map[string]string); ok {
			// For ParseSystemInfoStructured format
			if kvPair["key"] == searchKey {
				return kvPair["value"]
			}
		}
	}
	return ""
}

// ExtractBetween extracts the text between two characters/strings
// Example: ExtractBetween("Used RAM: 3.2Gi (42.1%)", "(", ")") returns "42.1%"
func ExtractBetween(str, start, end string) string {
	// Find the starting position
	startIdx := strings.Index(str, start)
	if startIdx == -1 {
		return "" // Start character not found
	}

	// Move past the start character
	startIdx += len(start)

	// Find the ending position after the start
	endIdx := strings.Index(str[startIdx:], end)
	if endIdx == -1 {
		return "" // End character not found
	}

	// Extract the substring
	return str[startIdx : startIdx+endIdx]
}

// increse number by 1
func IncNumber(i int) int {
	return i + 1
}

// ====================================================================== ========================== ======================================================================

// View provides a set of methods (e.g. render()) for rendering purpose.
func (c *Controller) View(t *template.Template, td *TemplateData, w http.ResponseWriter, r *http.Request) {

	uc := c.Config.GetValue("UnderConstruction")
	if uc != nil {
		if uc == true {

			InfoMessage("Site is under construction ... redirecting to Underconstruction page")
			InfoMessage("Remote Address: " + r.RemoteAddr)
			InfoMessage("X-Forwarded-For: " + r.Header.Get("X-Forwarded-For"))

			tmp := strings.Split(r.RemoteAddr, ":")
			rip := ""
			if len(tmp) > 0 {
				rip = tmp[0]
			}

			proxyIP := r.Header.Get("X-Forwarded-For")
			if rip == "127.0.0.1" && proxyIP != "" {
				rip = proxyIP
			}

			exipsval := c.Config.GetValue("ExcludeIPs")
			if exipsval != nil {
				exips := strings.Split(fmt.Sprint(exipsval), ",")
				if len(exips) > 0 {
					for _, ip := range exips {
						ip = strings.Trim(ip, " ")
						if ip == rip {
							InfoMessage("Request ip found in exclude list: " + ip)
							uc = false
						}
					}
				}
			}

			if uc == true {
				ut, err := c.GetUnderConstructionTemplate(c.UnderConstructionPage)
				if err != nil {
					ServerError(w, err)
					return
				}
				t = ut
			}
		}
	}

	/* Execute template */
	buf := new(bytes.Buffer)

	err := t.Execute(buf, td)
	if err != nil {
		ServerError(w, err)
		log.Fatal()
		return
	}

	_, err = buf.WriteTo(w)

	if err != nil {
		ServerError(w, err)
		log.Fatal()
		return
	}
}
