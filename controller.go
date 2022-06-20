package gomvc

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
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
	ActionCreate Action = 1
	ActionUpdate Action = 2
	ActionDelete Action = 3
	//ActionAuth   Action = 9
)

type Action int

var Session *scs.SessionManager
var Auth AuthObject

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
	next      string
	action    Action
	hasTable  bool
	needsAuth bool
}

type ActionRouting struct {
	URL       string
	NextURL   string
	NeedsAuth bool
	isWebHook bool
}

type RequestObject struct {
	baseUrl string
	cntrlr  string
	action  string
	params  map[string][]interface{}
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

// Controller URL key function
func (r *ActionRouting) getControllerOptionsKey(action Action) string {
	cKey := r.URL
	if strings.Contains(r.URL, "*") {
		cKey = strings.Replace(r.URL, "*", "", 1)
	}
	if strings.Contains(r.URL, "{id}") {
		cKey = strings.Replace(r.URL, "{id}", "", 1)
	}
	//cKey = cKey + "-" + fmt.Sprint(action)

	return cKey
}

// return session manager
func (c *Controller) GetSession() *scs.SessionManager {
	return Session
}

//return Authobject
func (c *Controller) GetAuthObject() *AuthObject {
	return &Auth
}

// register controller action - route, next, action and model
func (c *Controller) RegisterAction(route ActionRouting, action Action, model *Model) {
	if c.Router == nil {
		log.Fatal("Controller is not initialized")
		return
	}
	if c.Options == nil {
		c.Options = make(map[string]controllerOptions, 0)
	}
	if c.Models == nil {
		c.Models = make(map[string]*Model, 0)
	}

	hasTable := false
	cKey := route.getControllerOptionsKey(action)

	fmt.Println("Registering route :", route.URL, " -> ", cKey)

	if model != nil {
		if len(model.Fields) == 0 {
			err := model.InitModel(c.DB, model.TableName, model.PKField)
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

	c.Options[cKey] = controllerOptions{next: route.NextURL, action: action, hasTable: hasTable, needsAuth: route.NeedsAuth}

	if action == ActionView {
		c.Router.With(noSurf).Get(route.URL, c.viewAction)
	}
	if action == ActionCreate {
		c.Router.With(noSurf).Post(route.URL, c.createAction)
	}
	if action == ActionUpdate {
		c.Router.With(noSurf).Post(route.URL, c.updateAction)
	}
	if action == ActionDelete {
		c.Router.With(noSurf).Post(route.URL, c.deleteAction)
	}
}

func (c *Controller) RegisterAuthAction(authURL string, nextURL string, model *Model, authObject AuthObject) {
	if c.Router == nil {
		log.Fatal("Controller is not initialized")
		return
	}
	if model == nil {
		log.Fatal("AUth Controller needs model")
		return
	}
	if c.Options == nil {
		c.Options = make(map[string]controllerOptions, 0)
	}
	if c.Models == nil {
		c.Models = make(map[string]*Model, 0)
	}

	route := ActionRouting{URL: authURL, NeedsAuth: true}

	cKey := route.getControllerOptionsKey(9)
	authObject.authURL = authURL
	Auth = authObject

	fmt.Println("Registering Auth route :", route.URL, " -> ", cKey)

	if len(model.Fields) == 0 {
		err := model.InitModel(c.DB, model.TableName, model.PKField)
		if err != nil {
			err = errors.New("Error initializing Model for table : " + model.TableName + "\n" + err.Error())
			ServerError(nil, err)
			log.Fatal()
			return
		}
	}
	c.Models[cKey] = model

	c.Options[cKey] = controllerOptions{next: nextURL, action: 9, hasTable: true}

	// View
	c.Router.With(noSurf).Get(authURL, c.viewAction)

	// Post username / password / credentials
	c.Router.With(noSurf).Post(authURL, c.authAction)
}

//Register route -> responsible for processing requests and generating responses
func (c *Controller) RegisterCustomAction(route ActionRouting, method int, model *Model, f http.HandlerFunc) {
	if c.Router == nil {
		log.Fatal("Controller is not initialized")
		return
	}
	if c.Options == nil {
		c.Options = make(map[string]controllerOptions, 0)
	}
	if c.Models == nil {
		c.Models = make(map[string]*Model, 0)
	}

	hasTable := false
	cKey := route.getControllerOptionsKey(Action(method))

	if model != nil {
		if len(model.Fields) == 0 {
			err := model.InitModel(c.DB, model.TableName, model.PKField)
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

	c.Options[cKey] = controllerOptions{next: route.NextURL, action: 0, hasTable: hasTable}

	if method == HttpGET {
		if route.isWebHook {
			c.Router.Get(route.URL, f)
		} else {
			c.Router.With(noSurf).Get(route.URL, f)
		}

	}
	if method == HttpPOST {
		if route.isWebHook {
			c.Router.Post(route.URL, f)
		} else {
			c.Router.With(noSurf).Post(route.URL, f)
		}
	}
}

// Load template files
func (c *Controller) CreateTemplateCache(homePageFileName string, layoutTemplateFileName string) error {
	if c.Router == nil {
		log.Fatal("Controller is not initialized")
		return errors.New("Controller is not initialized")
	}
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

// AddTemplateData adds data for all templates
func (c *Controller) AddTemplateData(td TemplateData, r *http.Request) TemplateData {
	td.Flash = Session.PopString(r.Context(), "flash")
	td.Error = Session.PopString(r.Context(), "error")
	td.Warning = Session.PopString(r.Context(), "warning")

	td.CSRFToken = nosurf.Token(r)
	return td
}

func (c *Controller) GetTemplate(page string) (*template.Template, error) {
	to, ok := c.TemplateCache[page]
	if !ok {
		//template not found because link exists but template file not .. this is fatal error
		err := errors.New("could not get template from template cache")
		return nil, err
	}

	pagefilename := to.filename
	t, err := template.New(page).Funcs(functions).ParseFiles(pagefilename)
	if err != nil {
		return nil, err
	}

	t, err = t.ParseGlob("./web/templates/" + c.TemplateLayout)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// (string, string, string, map[string][]interface{})
func parseRequest(r *http.Request, homePageFilename string) RequestObject {
	rParts := strings.Split(r.URL.String(), "?")
	var params = make(map[string][]interface{}, 0)
	var retValue RequestObject

	cntrlr, action, paramsStr, baseUrl := exportControllerAndAction(rParts[0])
	if len(paramsStr) > 0 {
		params["***KEY***"] = []interface{}{paramsStr}
	}

	//Build params from url string [part 2]
	if len(rParts) > 1 {
		tmp2 := strings.Split(rParts[1], "&")

		for _, vv := range tmp2 {
			tmp3 := strings.SplitN(vv, "=", 2)
			if len(tmp3) > 1 {
				//fmt.Println(">1 : ", tmp3)
				var ppp = make(map[string]interface{}, 0)

				urlStr, err := url.QueryUnescape(tmp3[1])
				if err == nil {
					err := json.Unmarshal([]byte(urlStr), &ppp)
					if err == nil {
						params[tmp3[0]] = append(params[tmp3[0]], ppp)
					} else {
						params[tmp3[0]] = []interface{}{urlStr}
					}
				}
			} else {
				if len(vv) > 0 {
					params["id"] = []interface{}{vv}
				}
			}

		}
	}

	if action == "" {
		action = "view"
	}
	if len(cntrlr) == 0 {
		hp := strings.Split(homePageFilename, ".")
		cntrlr = hp[0]
	}

	retValue = RequestObject{baseUrl: baseUrl, cntrlr: cntrlr, action: action, params: params}

	return retValue
}

func exportControllerAndAction(urlFirstPart string) (string, string, string, string) {
	cntrlr, action, params, baseUrl := "", "", "", ""
	www := strings.Split(urlFirstPart, "/")
	for i, p := range www {
		//controller
		if i == 1 {
			cntrlr = strings.TrimSpace(p)
			baseUrl = "/" + cntrlr
		}
		//action
		if i == 2 {
			action = strings.TrimSpace(p)
			baseUrl = baseUrl + "/" + action
		}
		if i == 3 {
			params = strings.TrimSpace(p)
			baseUrl = baseUrl + "/"
		}
	}

	return cntrlr, action, params, baseUrl
}

func (c *Controller) authAction(w http.ResponseWriter, r *http.Request) {
	var err error

	Session.RenewToken(r.Context())

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options, URL : " + rObj.baseUrl)
		ServerError(w, err)
		return
	}

	m := c.Models[rObj.baseUrl]

	// Build filter -> only for primary key
	f := make([]Filter, 0)
	username := r.Form.Get(Auth.UsernameFieldName)
	password := r.Form.Get(Auth.PasswordFieldName)

	if len(username) == 0 || len(password) == 0 {
		err = errors.New("POST does not include credential values, URL : " + rObj.baseUrl)
		ServerError(w, err)
		return
	}

	f = append(f, Filter{Field: m.TableName + "." + Auth.UsernameFieldName, Operator: "=", Value: username})
	if len(Auth.ExtraConditions) > 0 {
		for _, v := range Auth.ExtraConditions {
			f = append(f, Filter{Field: v.Field, Operator: v.Operator, Value: v.Value, Logic: "AND"})
		}
	}

	//Get single row
	rr, err := m.GetRecords(f, 1)
	if err != nil {
		ServerError(w, err)
		return
	}

	if len(rr) > 0 {
		//fmt.Println(rr)
		//uIndx := rr[0].GetFieldIndex(cOptions.auth.UsernameFiledName)
		pIndx := rr[0].GetFieldIndex(Auth.PasswordFieldName)
		storedPass := fmt.Sprint(rr[0].Values[pIndx])
		idIndx := rr[0].GetFieldIndex(m.PKField)

		if Auth.CheckPasswordHash(password, storedPass) {
			token := Auth.TokenGenerator()

			// build fields
			var fields []SQLField
			fields = append(fields, SQLField{FieldName: Auth.HashCodeFieldName, Value: token})
			fields = append(fields, SQLField{FieldName: Auth.ExpTimeFieldName, Value: Auth.GetExpirationFromNow()})

			_, err = m.Update(fields, fmt.Sprint(rr[0].Values[idIndx]))

			if err != nil {
				ServerError(w, err)
				return
			}

			// Log messages
			InfoMessage("Logged in successful")
			if len(Auth.LoggedInMessage) > 0 {

				Session.Put(r.Context(), "flash", Auth.LoggedInMessage)
			}

			//store session token
			Session.Put(r.Context(), Auth.SessionKey, token)

			if len(cOptions.next) > 0 {
				http.Redirect(w, r, string(cOptions.next), http.StatusSeeOther)
			} else {
				c.viewAction(w, r)
			}
		} else {
			// wrong password
			InfoMessage("Login Fail")
			if len(Auth.LoginFailMessage) > 0 {
				Session.Put(r.Context(), "error", Auth.LoginFailMessage)
			}
			c.viewAction(w, r)
			return
		}
	} else {
		// user not found
		InfoMessage("Login Fail")
		if len(Auth.LoginFailMessage) > 0 {
			Session.Put(r.Context(), "error", Auth.LoginFailMessage)
		}
		c.viewAction(w, r)
		return
	}
}

// View Action --- GET ---
func (c *Controller) viewAction(w http.ResponseWriter, r *http.Request) {
	var rr []ResultRow
	var err error

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options, URL : " + rObj.baseUrl)
		ServerError(w, err)
		return
	}

	// Auth process
	if cOptions.needsAuth {
		if len(Auth.SessionKey) > 0 {

			exp, err := Auth.IsSessionExpired(r)
			if err != nil {
				ServerError(w, err)
				return
			}
			if exp {
				http.Redirect(w, r, Auth.authURL, http.StatusSeeOther)
			}
		}
	}

	if cOptions.hasTable {
		m := c.Models[rObj.baseUrl]
		if len(rObj.params) == 0 {
			// Get all rows
			rr, err = m.GetRecords([]Filter{}, 0)
			if err != nil {
				ServerError(w, err)
				return
			}
		} else {
			// Build filter -> only for primary key
			f := make([]Filter, 0)
			fv, ok := rObj.params["***KEY***"]
			if ok {
				f = append(f, Filter{Field: m.TableName + "." + m.PKField, Operator: "=", Value: fv[0]})
			}

			// Multiple filters -> ?filters={"name":"ford","description":"2021"}
			for k, v := range rObj.params {
				if k == "filters" {
					for _, vv := range v {
						vvMap, _ := vv.(map[string]interface{})
						for kkk, vvv := range vvMap {
							if FindInSlice(m.Fields, kkk) > -1 {
								if len(f) > 0 {
									f = append(f, Filter{Field: m.TableName + "." + kkk, Operator: " LIKE ", Value: "%" + vvv.(string) + "%", Logic: "AND"})
								} else {
									f = append(f, Filter{Field: m.TableName + "." + kkk, Operator: " LIKE ", Value: "%" + vvv.(string) + "%"})
								}

							}
						}
					}
				}
			}

			//Get single row
			rr, err = m.GetRecords(f, 1)
			if err != nil {
				ServerError(w, err)
				return
			}
		}
	}

	/* Get page template from name */
	page := rObj.cntrlr + "." + rObj.action + ".tmpl"

	InfoMessage(" - File : " + page + " - URL : " + rObj.baseUrl + " - Params : " + fmt.Sprint(rObj.params))

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
		t, err = c.GetTemplate(page)
		if err != nil {
			ServerError(w, err)
			return
		}
	}

	var td TemplateData
	td.Result = rr
	td.URLParams = rObj.params
	m, ok := c.Models[rObj.baseUrl]
	if ok {
		td.Model = m.Instance()
	}

	td = c.AddTemplateData(td, r)

	View(t, w, &td)
}

// Create Action --- POST ---
func (c *Controller) createAction(w http.ResponseWriter, r *http.Request) {
	var err error

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, hasOptions := c.Options[rObj.baseUrl]
	if !hasOptions {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}

	// Auth process
	if cOptions.needsAuth {
		if len(Auth.SessionKey) > 0 {

			exp, err := Auth.IsSessionExpired(r)
			if err != nil {
				ServerError(w, err)
				return
			}
			if exp {
				http.Redirect(w, r, Auth.authURL, http.StatusSeeOther)
			}
		}
	}

	if !cOptions.hasTable {
		err = errors.New("this action (createAction) needs a database table")
		ServerError(w, err)
		return
	}

	m, ok := c.Models[rObj.baseUrl]
	if !ok {
		err = errors.New("Model for controller : " + rObj.baseUrl + " not found")
		ServerError(w, err)
		return
	}

	var fields []SQLField

	for _, f := range m.Fields {
		var fv = r.Form.Get(f)
		if fv != "" {
			fields = append(fields, SQLField{FieldName: f, Value: fv})
		}
	}

	InfoMessage("Starting Create process !!!")

	_, err = m.Save(fields)
	if err != nil {
		ServerError(w, err)
		return
	}

	if len(cOptions.next) > 0 {
		http.Redirect(w, r, string(cOptions.next), http.StatusSeeOther)
	} else {
		c.viewAction(w, r)
	}
}

// Update Action --- POST ---
func (c *Controller) updateAction(w http.ResponseWriter, r *http.Request) {
	var err error

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}

	// Auth process
	if cOptions.needsAuth {
		if len(Auth.SessionKey) > 0 {

			exp, err := Auth.IsSessionExpired(r)
			if err != nil {
				ServerError(w, err)
				return
			}
			if exp {
				http.Redirect(w, r, Auth.authURL, http.StatusSeeOther)
			}
		}
	}

	if !cOptions.hasTable {
		err = errors.New("this action (updateAction) needs a database table")
		ServerError(w, err)
		return
	}

	m, ok := c.Models[rObj.baseUrl]
	if !ok {
		err = errors.New("Model for controller : " + rObj.baseUrl + " not found")
		ServerError(w, err)
		return
	}
	var fields []SQLField

	for _, f := range m.Fields {
		var fv = r.Form.Get(f)
		if fv != "" {
			fields = append(fields, SQLField{FieldName: f, Value: fv})
		}
	}

	InfoMessage("Starting Update process !!!")

	id, ok := rObj.params["***KEY***"]
	if ok {
		_, err = m.Update(fields, fmt.Sprint(id[0]))
		if err != nil {
			ServerError(w, err)
			return
		}
	} else {
		err = errors.New("Table's primary key [" + m.PKField + "] not found in parameters array." +
			"Url parameters must have [" + m.PKField + "] as parameter OR table must have [id] field as primary key")
		ServerError(w, err)
		return
	}

	if len(cOptions.next) > 0 {
		http.Redirect(w, r, cOptions.next, http.StatusSeeOther)
	} else {
		c.viewAction(w, r)
	}
}

// Delete Action --- POST ---
func (c *Controller) deleteAction(w http.ResponseWriter, r *http.Request) {

	var err error

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options")
		ServerError(w, err)
		return
	}

	// Auth process
	if cOptions.needsAuth {
		if len(Auth.SessionKey) > 0 {

			exp, err := Auth.IsSessionExpired(r)
			if err != nil {
				ServerError(w, err)
				return
			}
			if exp {
				http.Redirect(w, r, Auth.authURL, http.StatusSeeOther)
			}
		}
	}

	if !cOptions.hasTable {
		err = errors.New("this action (updateAction) needs a database table")
		ServerError(w, err)
		return
	}

	m, ok := c.Models[rObj.baseUrl]
	if !ok {
		err = errors.New("Model for controller : " + rObj.baseUrl + " not found")
		ServerError(w, err)
		return
	}

	InfoMessage("Starting Delete process !!!")

	id, ok := rObj.params["***KEY***"]
	if ok {
		_, err = m.Delete(fmt.Sprint(id[0]))
		if err != nil {
			ServerError(w, err)
			return
		}
	} else {
		err = errors.New("Table's primary key [" + m.PKField + "] not found in parameters array." +
			"Url parameters must have [" + m.PKField + "] as parameter OR table must have [id] field as primary key")
		ServerError(w, err)
		return
	}

	if len(cOptions.next) > 0 {
		http.Redirect(w, r, cOptions.next, http.StatusSeeOther)
	} else {
		c.viewAction(w, r)
	}
}
