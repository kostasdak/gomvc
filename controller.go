package gomvc

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/justinas/nosurf"
)

const (
	HttpGET  int = 0
	HttpPOST int = 1
)

const (
	ActionView   Action = 0
	ActionCreate        = 1
	ActionUpdate        = 2
	ActionDelete        = 3
)

type Action int

var Session *scs.SessionManager

type Controller struct {
	DB               *sql.DB
	Models           map[string]*Model
	TemplateCache    map[string]TemplateObject
	TemplateLayout   string
	TemplateHomePage string
	Options          map[string]controllerOptions
	Router           *chi.Mux
	Config           *AppConfig
}

type controllerOptions struct {
	next     string
	action   Action
	method   int
	hasTable bool
}

type TemplateObject struct {
	filename string
	template *template.Template
}

type TableObject struct {
	MainTable     string
	RelatedTables string
}

var functions = template.FuncMap{}

// Pass pointer to db connection and appconfig struct
func (c *Controller) Initialize(db *sql.DB, cfg *AppConfig) {
	c.DB = db
	c.Config = cfg
	c.Router = chi.NewRouter()
	c.Router.Use(noSurf)

	Session = scs.New()
	Session.Lifetime = 24 * time.Hour
	Session.Cookie.Persist = true
	Session.Cookie.SameSite = http.SameSiteLaxMode
	Session.Cookie.Secure = c.Config.Server.SessionSecure

	c.Router.Use(sessionLoad)

	InitHelpers(c.Config)
}

// noSurf midleware ... is the csrf protection middleware
func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	return csrfHandler
}

// session midleware ... is the session middleware
func sessionLoad(next http.Handler) http.Handler {
	return Session.LoadAndSave(next)
}

func (c *Controller) GetSession() *scs.SessionManager {
	return Session
}

func (c *Controller) RegisterAction(route string, next string, action Action, model *Model) {
	if c.Options == nil {
		c.Options = make(map[string]controllerOptions, 0)
	}
	if c.Models == nil {
		c.Models = make(map[string]*Model, 0)
	}

	hasTable := false
	cKey := strings.Replace(route, "*", "", 1)

	if model != nil {
		if len(model.Fields) == 0 {
			err := model.InitModel(c.DB, model.TableName, "id")
			if err != nil {
				err = errors.New("Error initializing Model for table : " + model.TableName + "\n" + err.Error())
				ServerError(nil, err)
				log.Fatal()
				return
			}
		}
		c.Models[cKey] = model

		hasTable = true
	}

	c.Options[cKey] = controllerOptions{next: next, action: action, hasTable: hasTable}
	//fmt.Println(key)

	if action == ActionView {
		c.Router.Get(route, c.viewAction)
	}
	if action == ActionCreate {
		c.Router.Post(route, c.createAction)
	}
	if action == ActionUpdate {
		c.Router.Post(route, c.updateAction)
	}
	if action == ActionDelete {
		c.Router.Post(route, c.deleteAction)
	}

}

//Register route -> responsible for processing requests and generating responses
func (c *Controller) RegisterCustomAction(route string, next string, method int, model *Model, f http.HandlerFunc) {
	if c.Options == nil {
		c.Options = make(map[string]controllerOptions, 0)
	}
	if c.Models == nil {
		c.Models = make(map[string]*Model, 0)
	}

	hasTable := false
	cKey := strings.Replace(route, "*", "", 1)

	if model != nil {
		if len(model.Fields) == 0 {
			err := model.InitModel(c.DB, model.TableName, "id")
			if err != nil {
				err = errors.New("Error initializing Model for table : " + model.TableName + "\n" + err.Error())
				ServerError(nil, err)
				log.Fatal()
				return
			}
		}

		c.Models[cKey] = model

		hasTable = true
	}

	c.Options[cKey] = controllerOptions{next: next, action: 0, method: method, hasTable: hasTable}

	if method == HttpGET {
		c.Router.Get(route, f)
	}
	if method == HttpPOST {
		c.Router.Post(route, f)
	}
}

// Load template files
func (c *Controller) CreateTemplateCache(homePageFileName string, layoutTemplateFileName string) error {
	myCache := make(map[string]TemplateObject, 0)
	c.TemplateLayout = layoutTemplateFileName
	c.TemplateHomePage = homePageFileName

	pages, err := filepath.Glob("./web/templates/*.tmpl")
	if err != nil {
		ServerError(nil, err)
		log.Fatal()
		return err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		fmt.Println("Loading page : " + page + " / name index : " + name)
		ts, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			err = errors.New("page file not found : " + page + "\n" + err.Error())
			ServerError(nil, err)
			log.Fatal()
			return err
		}

		ts, err = ts.ParseGlob("./web/templates/" + layoutTemplateFileName)
		if err != nil {
			err = errors.New("layout file not found : " + page + "\n" + err.Error())
			ServerError(nil, err)
			log.Fatal()
			return err
		}

		myCache[name] = TemplateObject{template: ts, filename: page}

	}

	c.TemplateCache = myCache
	return nil
}

// addTemplateData adds data for all templates
func (c *Controller) addTemplateData(td TemplateData, r *http.Request) TemplateData {
	td.Flash = Session.PopString(r.Context(), "flash")
	td.Error = Session.PopString(r.Context(), "error")
	td.Warning = Session.PopString(r.Context(), "warning")

	td.CSRFToken = nosurf.Token(r)
	return td
}

func extractUrlPath(r *http.Request, homePageFile string) (string, string, string, []string) {
	var cntrlr string
	var action string
	var params []string
	/* Extract Controller / View / Params */
	www := strings.Split(r.URL.String(), "/")
	baseUrl := ""

	hp := strings.Split(homePageFile, ".")

	for i, p := range www {
		if i == 1 {
			cntrlr = strings.TrimSpace(p)
			baseUrl = "/" + cntrlr
		}
		if i == 2 {
			action = strings.TrimSpace(p)
			baseUrl = baseUrl + "/" + action
		}
		if i > 2 {

			tmp := strings.Split(p, "?")

			if len(tmp) > 1 {
				for _, v := range tmp {
					tmp2 := strings.Split(v, "&")

					if len(tmp2) > 1 {

						params = append(params, tmp2...)

					} else {

						params = append(params, v)

					}
				}
			} else {
				params = append(params, p)
			}

			baseUrl = baseUrl + "/"
		}
	}
	if action == "" {
		action = "view"
	}
	if len(cntrlr) == 0 {
		cntrlr = hp[0]
	}

	return baseUrl, cntrlr, action, params
}

// View Action --- GET ---
func (c *Controller) viewAction(w http.ResponseWriter, r *http.Request) {
	var rr []ResultRow
	var err error

	baseUrl, cntrlr, action, params := extractUrlPath(r, c.TemplateHomePage)

	cOptions, ok := c.Options[baseUrl]
	if !ok {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}

	if cOptions.hasTable {
		if len(params) == 0 {
			//load all models
			rr, err = c.Models[baseUrl].GetAllRecords(0)
			if err != nil {
				ServerError(w, err)
				return
			}
			//related result ?

		} else {
			//load single model
			id, _ := strconv.ParseInt(params[0], 10, 64)
			rr = make([]ResultRow, 1)
			rr[0], err = c.Models[baseUrl].GetRecordByPK(id)
			if err != nil {
				ServerError(w, err)
				return
			}
			//related result ?
		}
	}

	/* Get page template from name */
	page := cntrlr + "." + action + ".tmpl"

	InfoMessage("Action View : " + page + " params : " + strings.Join(params, ","))

	var t *template.Template
	if c.Config.UseCache {
		to, ok := c.TemplateCache[page]
		if !ok {
			//template not found because link exists but template file not .. this is fatal error
			err = errors.New("could not get template from template cache")
			ServerError(w, err)
			return
		}
		t = to.template
	} else {
		to, ok := c.TemplateCache[page]
		if !ok {
			//template not found because link exists but template file not .. this is fatal error
			err = errors.New("could not get template from template cache")
			ServerError(w, err)
			return
		}

		pagefilename := to.filename
		t, err = template.New(page).Funcs(functions).ParseFiles(pagefilename)
		if err != nil {
			ServerError(w, err)
			return
		}

		t, err = t.ParseGlob("./web/templates/" + c.TemplateLayout)
		if err != nil {
			ServerError(w, err)
			return
		}
	}

	var td TemplateData
	td.Result = rr
	m, ok := c.Models[baseUrl]
	if ok {
		td.Model = m.Instance()
	}

	td = c.addTemplateData(td, r)

	View(t, w, &td)
}

// Create Action --- POST ---
func (c *Controller) createAction(w http.ResponseWriter, r *http.Request) {
	var m Model
	var err error
	baseUrl, cntrlr, _, _ := extractUrlPath(r, c.TemplateHomePage)

	cOptions, ok := c.Options[baseUrl]
	if !ok {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}
	if !cOptions.hasTable {
		err = errors.New("this action (createAction) needs a database table")
		ServerError(w, err)
		return
	}

	m.InitModel(c.DB, cntrlr, "id")

	var vals = make(map[string]string)

	for _, f := range m.Fields {
		var fv = r.Form.Get(f)
		if fv != "" {
			vals[f] = fv
		}
	}

	InfoMessage("Starting Create process !!!")

	m.Save(vals)

	if ok {
		http.Redirect(w, r, string(cOptions.next), http.StatusSeeOther)
	} else {
		c.viewAction(w, r)
	}
}

// Update Action --- POST ---
func (c *Controller) updateAction(w http.ResponseWriter, r *http.Request) {
	var m Model
	var err error

	baseUrl, cntrlr, _, params := extractUrlPath(r, c.TemplateHomePage)

	cOptions, ok := c.Options[baseUrl]
	if !ok {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}
	if !cOptions.hasTable {
		err = errors.New("this action (updateAction) needs a database table")
		ServerError(w, err)
		return
	}

	m.InitModel(c.DB, cntrlr, "id")

	var vals = make(map[string]string)

	for _, f := range m.Fields {
		var fv = r.Form.Get(f)
		if fv != "" {
			vals[f] = fv
		}
	}

	InfoMessage("Starting Update process !!!")

	m.Update(vals, params[0])

	if ok {
		http.Redirect(w, r, cOptions.next, http.StatusSeeOther)
	} else {
		c.viewAction(w, r)
	}
}

//Delete Action --- POST ---
func (c *Controller) deleteAction(w http.ResponseWriter, r *http.Request) {
	var m Model
	var err error

	baseUrl, cntrlr, _, params := extractUrlPath(r, c.TemplateHomePage)

	cOptions, ok := c.Options[baseUrl]
	if !ok {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}
	if !cOptions.hasTable {
		err = errors.New("this action (deleteAction) needs a database table")
		ServerError(w, err)
		return
	}

	m.InitModel(c.DB, cntrlr, "id")

	InfoMessage("Starting Delete process !!!")

	m.Delete(params[0])

	if ok {
		http.Redirect(w, r, cOptions.next, http.StatusSeeOther)
	} else {
		c.viewAction(w, r)
	}
}
