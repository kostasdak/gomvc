package gomvc

import (
	"bytes"
	"log"
	"net/http"
	"text/template"
)

type TemplateData struct {
	/*StringMap map[string]string
	IntMap    map[string]int
	FloatMap  map[string]float32
	Data      map[string]interface{}*/
	Model     Model
	Result    []ResultRow
	CSRFToken string
	Flash     string
	Warning   string
	Error     string
}

//View provides a set of methods (e.g. render()) for rendering purpose.
func View(t *template.Template, w http.ResponseWriter, td *TemplateData) {
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
