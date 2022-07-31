package gomvc

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
)

type TemplateData struct {
	Model        Model
	Result       []ResultRow
	URLParams    map[string][]interface{}
	CustomValues map[string][]interface{}
	CSRFToken    string
	Flash        string
	Warning      string
	Error        string
}

//View provides a set of methods (e.g. render()) for rendering purpose.
func (c *Controller) View(t *template.Template, td *TemplateData, w http.ResponseWriter, r *http.Request) {

	uc := c.Config.GetValue("UnderConstruction")
	if uc != nil {
		if uc == true {

			InfoMessage("Site is under construction ... redirecting to Underconstruction page")
			InfoMessage("Remote Address : " + r.RemoteAddr)
			InfoMessage("X-Forwarded-For : " + r.Header.Get("X-Forwarded-For"))

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
							InfoMessage("Request ip found in exclude list : " + ip)
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
