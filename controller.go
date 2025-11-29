// Package gomvc is a Golang package easy to use and build almost any MVC Web App connected to MySql database with just a few steps.
// `gomvc` package requires a MySql Server up and running and a database ready to drive your web application.
//
// Build a standard MVC (Model, View, Controller) style web app with minimum Golang code, like you use a classic MVC Framework.
// Many features, many ready to use functions, highly customizable, embeded log and error handling
//
// #### MVC
//
// ```
// (databse CRUD)      (http req/resp)
//
//	Model <--------> Controller
//	    \            /
//	     \          /
//	      \        /
//	       \      /
//	        \    /
//	         View
//	 (text/template files)
//
// ```
//
// #### Basic Steps
// * Edit the config file
// * Load config file `config.yaml`
// * Connect to MySql database
// * Write code to initialize your Models and Controllers
// * Write your standard text/Template files (Views)
// * Start your server and enjoy
//
// #### More Examples
// Find mire examples in Readme.md file
package gomvc

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
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

// HttpGET, HttpPOST constants are helping the use of the package when it comes to the type of request
const (
	HttpGET  int = 0
	HttpPOST int = 1
)

const (
	ActionView   Action = 0
	ActionCreate Action = 1
	ActionUpdate Action = 2
	ActionDelete Action = 3
)

// Action defines the type of action to execute from a handler.
// ActionVew = return data to http client
// ActionCreate, ActionUpdate, ActionDelete = create, update, delete records from database,
// this action are more likeky to accompaned with an ActionView action so they return a result to the http client after the action
type Action int

// Session is the SessionManager that will work as a middleware.
var Session *scs.SessionManager

// Auth is the authentication object
var Auth AuthObject

// Controller is the controller struct, contains the models, the templates, the web layout, the home page, the under construction page
// the controller options for each route, the router itself and the config struct.
type Controller struct {
	DB                      *sql.DB
	Models                  map[string]*Model
	TemplateCache           map[string]TemplateObject
	TemplateLayout          string
	TemplateHomePage        string
	UnderConstructionLayout string
	UnderConstructionPage   string
	Options                 map[string]controllerOptions
	Router                  *chi.Mux
	Config                  *AppConfig
	Functions               template.FuncMap

	IPRateLimiter   *RateLimiter // Rate limit by IP
	UserRateLimiter *RateLimiter // Rate limit by username
}

// controllerOptions is a struct that holds options for each route in Controller
type controllerOptions struct {
	next      string
	action    Action
	hasTable  bool
	needsAuth bool
}

// ActionRouting helps the router to have the routing information about the URL, the NextURL,
// if the route needs authentication or if it is a web hook (web hook can have POST data without midleware CSRF check)
type ActionRouting struct {
	URL       string
	NextURL   string
	NeedsAuth bool
	IsWebHook bool
}

// RequestObject is a struct builded from the http request, holds the url data in a convinient way.
type RequestObject struct {
	baseUrl string
	cntrlr  string
	action  string
	params  map[string][]interface{}
}

// TemplateObject is the template struct, holds the filename and the template object.
type TemplateObject struct {
	filename string
	template *template.Template
}

// Build func map
//var Functions = template.FuncMap{}

// Initialize from this function we pass a pointer to db connection and a pointer to appconfig struct
func (c *Controller) Initialize(db *sql.DB, cfg *AppConfig) {
	c.DB = db
	c.Config = cfg
	c.Router = chi.NewRouter()

	InitHelpers(c.Config)

	// Firewall ditection and help
	if cfg.Server.Port > 0 {
		DisplayFirewallHelp(cfg.Server.Port)
	}

	// Initialize rate limiters if enabled
	if cfg.RateLimit.Enabled {
		if cfg.RateLimit.IPMaxAttempts > 0 && cfg.RateLimit.IPBlockMinutes > 0 {
			c.IPRateLimiter = NewRateLimiter(
				cfg.RateLimit.IPMaxAttempts,
				time.Minute*time.Duration(cfg.RateLimit.IPBlockMinutes),
			)
			InfoMessage(fmt.Sprintf("IP Rate Limiting enabled: %d attempts, %d minute block",
				cfg.RateLimit.IPMaxAttempts, cfg.RateLimit.IPBlockMinutes))
		}

		if cfg.RateLimit.UsernameMaxAttempts > 0 && cfg.RateLimit.UsernameBlockMinutes > 0 {
			c.UserRateLimiter = NewRateLimiter(
				cfg.RateLimit.UsernameMaxAttempts,
				time.Minute*time.Duration(cfg.RateLimit.UsernameBlockMinutes),
			)
			InfoMessage(fmt.Sprintf("Username Rate Limiting enabled: %d attempts, %d minute block",
				cfg.RateLimit.UsernameMaxAttempts, cfg.RateLimit.UsernameBlockMinutes))
		}
	} else {
		InfoMessage("Rate limiting is disabled")
	}

	Session = scs.New()
	Session.Lifetime = 24 * time.Hour
	Session.Cookie.Persist = true
	Session.Cookie.SameSite = http.SameSiteLaxMode
	Session.Cookie.Secure = true // Always Secure Cookie as default

	// Set Secure flag based on environment
	// In production/staging, require secure cookies
	// In development, allow non-secure for HTTP testing
	if c.Config.Server.SessionSecure {
		Session.Cookie.Secure = true
	} else {
		Session.Cookie.Secure = false
		InfoMessage("Development mode: Session cookies are NOT secure (HTTP allowed)")
	}

	// Add security middleware with environment awareness
	c.Router.Use(secureHeaders(cfg))

	c.Router.Use(sessionLoad)
}

// noSurf midleware ... is the CSRF protection middleware
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

// sessionLoad session midleware function
func sessionLoad(next http.Handler) http.Handler {
	return Session.LoadAndSave(next)
}

// secureHeaders middleware adds security headers and enforces HTTPS based on environment
func secureHeaders(cfg *AppConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isProduction := cfg.Server.SessionSecure
			//isStaging := cfg.Environment == "staging"
			isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

			// Enforce HTTPS only in production
			if isProduction && !isHTTPS {
				httpsURL := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}

			// Always set basic security headers (safe for all environments)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Only set strict security headers in production
			if isProduction {
				w.Header().Set("X-Frame-Options", "DENY")
				w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

				// HSTS header (only when using HTTPS)
				if isHTTPS {
					maxAge := "31536000" // 1 year
					w.Header().Set("Strict-Transport-Security",
						"max-age="+maxAge+"; includeSubDomains; preload")
				}

				// Strict CSP for production
				w.Header().Set("Content-Security-Policy",
					"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';")
			} else {
				// Development environment - relaxed headers
				w.Header().Set("X-Frame-Options", "SAMEORIGIN")
				w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")

				// Relaxed CSP for development (allows more flexibility)
				w.Header().Set("Content-Security-Policy",
					"default-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https: http:; font-src 'self' data:;")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getControllerOptionsKey get the (key or id) from URL, Controller key function
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

// GetSession return session manager
func (c *Controller) GetSession() *scs.SessionManager {
	return Session
}

// GetAuthObject return Authobject
func (c *Controller) GetAuthObject() *AuthObject {
	return &Auth
}

// RegisterAction register controller action - route, next, action and model
// RegisterAction, RegisterAuthAction, RegisterCustomAction are the most important functions in the gomvc package
// all functions are responsible for processing requests and generating responses.
// RegisterAction is used to register one of the pre defined actions View, Create, Update, Delete
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

	// Show log message
	if len(c.Options) == 0 {
		fmt.Println("")
		InfoMessage(CenterText("REGISTERING ROUTER ACTIONS", 40, '='))
	}

	hasTable := false
	cKey := route.getControllerOptionsKey(action)

	fmt.Println("Registering route :", route.URL, " -> ", cKey)

	if model != nil {
		if len(model.Fields) == 0 {
			err := model.InitModel(c.DB, model.TableName, model.PKField)
			if err != nil {
				err = errors.New("Error initializing Model for table: " + model.TableName + "\n" + err.Error())
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

// RegisterCustomAction register controller action - route, next, action and model
// RegisterAction, RegisterAuthAction, RegisterCustomAction are the most important functions in the gomvc package
// all functions are responsible for processing requests and generating responses.
// RegisterCustomAction is used to register any custom action that doesn't fit the pre defined actions View, Create, Update, Delete
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

	// Show log message
	if len(c.Options) == 0 {
		fmt.Println("")
		InfoMessage(CenterText("REGISTERING ROUTER ACTIONS", 40, '='))
	}

	hasTable := false
	cKey := route.getControllerOptionsKey(Action(method))

	fmt.Println("Registering route :", route.URL, " -> ", cKey)

	if model != nil {
		if len(model.Fields) == 0 {
			err := model.InitModel(c.DB, model.TableName, model.PKField)
			if err != nil {
				err = errors.New("Error initializing Model for table: " + model.TableName + "\n" + err.Error())
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
		if route.IsWebHook {
			c.Router.Get(route.URL, f)
		} else {
			c.Router.With(noSurf).Get(route.URL, f)
		}

	}
	if method == HttpPOST {
		if route.IsWebHook {
			c.Router.Post(route.URL, f)
		} else {
			c.Router.With(noSurf).Post(route.URL, f)
		}
	}
}

// RegisterAuthAction register controller action - route, next, action and model
// is used to register the authentication actions
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

	// Show log message
	if len(c.Options) == 0 {
		fmt.Println("")
		InfoMessage(CenterText("REGISTERING ROUTER ACTIONS", 40, '='))
	}

	route := ActionRouting{URL: authURL, NeedsAuth: true}

	cKey := route.getControllerOptionsKey(9)
	authObject.authURL = authURL
	Auth = authObject

	fmt.Println("Registering Auth route:", route.URL, " -> ", cKey)

	if len(model.Fields) == 0 {
		err := model.InitModel(c.DB, model.TableName, model.PKField)
		if err != nil {
			err = errors.New("Error initializing Model for table: " + model.TableName + "\n" + err.Error())
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

// RegisterAuthActionLinux register controller action - route, next, action and model
// is used to register the authentication actions
func (c *Controller) RegisterAuthActionLinux(authURL string, nextURL string, authObject AuthObject) {
	if c.Router == nil {
		log.Fatal("Controller is not initialized")
		return
	}

	if c.Options == nil {
		c.Options = make(map[string]controllerOptions, 0)
	}

	// Show log message
	if len(c.Options) == 0 {
		fmt.Println("")
		InfoMessage(CenterText("REGISTERING ROUTER ACTIONS", 40, '='))
	}

	route := ActionRouting{URL: authURL, NeedsAuth: true}

	cKey := route.getControllerOptionsKey(9)
	authObject.authURL = authURL
	Auth = authObject

	fmt.Println("Registering Auth route:", route.URL, " -> ", cKey)

	c.Options[cKey] = controllerOptions{next: nextURL, action: 9, hasTable: false}

	// View
	c.Router.With(noSurf).Get(authURL, c.viewAction)

	// Post username / password / credentials
	c.Router.With(noSurf).Post(authURL, c.authActionLinux)
}

// CreateTemplateCache loads the template files and creates a cache of templates in controller.
func (c *Controller) CreateTemplateCache(homePageFileName string, layoutTemplateFileName string) error {
	fmt.Println("")

	InfoMessage(CenterText("READING TEMPLATE FILES", 40, '='))

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
		fmt.Println("Loading page: " + page + " / name index: " + name)
		ts, err := template.New(name).Funcs(c.Functions).ParseFiles(page)
		if err != nil {
			err = errors.New("page file not found: " + page + "\n" + err.Error())
			ServerError(nil, err)
			log.Fatal()
			return err
		}

		ts, err = ts.ParseGlob("./web/templates/" + layoutTemplateFileName)
		if err != nil {
			err = errors.New("layout file not found: " + page + "\n" + err.Error())
			ServerError(nil, err)
			log.Fatal()
			return err
		}

		myCache[name] = TemplateObject{template: ts, filename: page}
	}

	c.TemplateCache = myCache

	return nil
}

// AddTemplateData adds data for templates, the data will be available in the view to build the web page before response.
func (c *Controller) AddTemplateData(td TemplateData, r *http.Request) TemplateData {
	td.Flash = Session.PopString(r.Context(), "flash")
	td.Error = Session.PopString(r.Context(), "error")
	td.Warning = Session.PopString(r.Context(), "warning")

	td.CSRFToken = nosurf.Token(r)
	return td
}

// GetTemplate return a single template from template cache
func (c *Controller) GetTemplate(page string) (*template.Template, error) {
	to, ok := c.TemplateCache[page]
	if !ok {
		//template not found because link exists but template file not .. this is fatal error
		err := errors.New("could not get template from template cache")
		return nil, err
	}

	pagefilename := to.filename
	t, err := template.New(page).Funcs(c.Functions).ParseFiles(pagefilename)
	if err != nil {
		return nil, err
	}

	// Layout file
	t, err = t.ParseGlob("./web/templates/" + c.TemplateLayout)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// GetFunctions returns the function map from the controller
func (c *Controller) GetFunctions() template.FuncMap {
	return c.Functions
}

// GetUnderConstructionTemplate get the under construction page
func (c *Controller) GetUnderConstructionTemplate(page string) (*template.Template, error) {
	to, ok := c.TemplateCache[page]
	if !ok {
		//template not found because link exists but template file not .. this is fatal error
		err := errors.New("could not get UnderConstruction template from template cache")
		return nil, err
	}

	pagefilename := to.filename
	t, err := template.New(page).Funcs(c.Functions).ParseFiles(pagefilename)
	if err != nil {
		return nil, err
	}

	// Layout file
	t, err = t.ParseGlob("./web/templates/" + c.UnderConstructionLayout)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// parseRequest parse request and build a RequestObject (string, string, string, map[string][]interface{})
func parseRequest(r *http.Request, homePageFilename string) RequestObject {
	rParts := strings.Split(r.URL.String(), "?")
	var params = make(map[string][]interface{}, 0)
	var retValue RequestObject

	InfoMessage("ParseRequest - URL: " + r.URL.String())

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

// exportControllerAndAction splits the url to parts and returns the controller name, the action name, and parameters
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

// authAction is the authentication function
func (c *Controller) authAction(w http.ResponseWriter, r *http.Request) {
	var err error

	Session.RenewToken(r.Context())

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options, URL: " + rObj.baseUrl)
		ServerError(w, err)
		return
	}

	m := c.Models[rObj.baseUrl]

	// Check IP-based rate limit FIRST
	// Get client IP
	clientIP := getClientIP(r)

	if c.IPRateLimiter != nil {
		if c.IPRateLimiter.IsBlocked(clientIP) {
			//blockedUntil := c.IPRateLimiter.GetBlockedUntil(clientIP)
			InfoMessage("Login attempt from blocked IP: " + clientIP)

			// Generic error message (don't reveal rate limiting)
			if len(Auth.LoginFailMessage) > 0 {
				Session.Put(r.Context(), "error", Auth.LoginFailMessage)
			}

			// Optional: Set a more specific message
			Session.Put(r.Context(), "error",
				"Too many failed attempts. Please try again later.")

			// Add delay to further slow down attackers
			time.Sleep(time.Second * 2)

			c.viewAction(w, r)
			return
		}
	}

	// Validate credentials are present
	username := r.Form.Get(Auth.UsernameFieldName)
	password := r.Form.Get(Auth.PasswordFieldName)

	if len(username) == 0 || len(password) == 0 {
		// Add delay to prevent timing leak
		time.Sleep(time.Millisecond * time.Duration(200+rand.Intn(100)))
		InfoMessage("Login failed: missing credentials")
		if len(Auth.LoginFailMessage) > 0 {
			Session.Put(r.Context(), "error", Auth.LoginFailMessage)
		}
		c.viewAction(w, r)
		return
	}

	// Check username-based rate limit
	if c.UserRateLimiter != nil {
		if c.UserRateLimiter.IsBlocked(username) {
			//blockedUntil := c.UserRateLimiter.GetBlockedUntil(username)
			InfoMessage("Login attempt for blocked username: " + username)

			// Record IP attempt too
			if c.IPRateLimiter != nil {
				c.IPRateLimiter.RecordFailedAttempt(clientIP)
			}

			if len(Auth.LoginFailMessage) > 0 {
				Session.Put(r.Context(), "error", Auth.LoginFailMessage)
			}

			time.Sleep(time.Second * 2)
			c.viewAction(w, r)
			return
		}
	}

	// Build filter for username lookup
	f := make([]Filter, 0)
	f = append(f, Filter{Field: m.TableName + "." + Auth.UsernameFieldName, Operator: "=", Value: username})
	if len(Auth.ExtraConditions) > 0 {
		for _, v := range Auth.ExtraConditions {
			f = append(f, Filter{Field: v.Field, Operator: v.Operator, Value: v.Value, Logic: "AND"})
		}
	}

	//Get single row [user record]
	rr, err := m.GetRecords(f, 1)
	if err != nil {
		ServerError(w, err)
		return
	}

	// TIMING ATTACK PREVENTION:
	// Always perform the same operations regardless of whether user exists
	// This prevents attackers from determining valid usernames by timing

	var storedPasswordHash string
	var userExists bool = len(rr) > 0
	var userID string

	if userExists {
		//fmt.Println(rr)
		//uIndx := rr[0].GetFieldIndex(cOptions.auth.UsernameFiledName)
		pIndx := rr[0].GetFieldIndex(Auth.PasswordFieldName)
		if pIndx == -1 {
			ServerError(w, errors.New("password field not found in user record"))
			return
		}

		storedPasswordHash = fmt.Sprint(rr[0].Values[pIndx])
		idIndx := rr[0].GetFieldIndex(m.PKField)
		if idIndx == -1 {
			ServerError(w, errors.New("primary key field not found in user record"))
			return
		}
		userID = fmt.Sprint(rr[0].Values[idIndx])
	} else {
		// Use a dummy hash to ensure bcrypt comparison still runs
		// This hash was generated with bcrypt.GenerateFromPassword([]byte("dummy"), 12)
		storedPasswordHash = "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW"
		userID = ""
	}

	// Always verify the password (even with dummy hash if user doesn't exist)
	// This ensures constant time regardless of username validity
	passwordValid := Auth.CheckPasswordHash(password, storedPasswordHash)

	// Only proceed if BOTH user exists AND password is valid
	if userExists && passwordValid {
		// SUCCESS
		// Reset rate limits on successful login
		if c.IPRateLimiter != nil {
			c.IPRateLimiter.ResetAttempts(clientIP)
		}
		if c.UserRateLimiter != nil {
			c.UserRateLimiter.ResetAttempts(username)
		}

		token := Auth.TokenGenerator()
		InfoMessage("Auth successful for user: " + username + " from IP: " + clientIP)

		// Build fields for session storage
		var exp time.Time = Auth.GetExpirationFromNow()
		var fields []SQLField
		fields = append(fields, SQLField{FieldName: Auth.HashCodeFieldName, Value: token})
		fields = append(fields, SQLField{FieldName: Auth.ExpTimeFieldName, Value: exp})

		// Update user record with session token
		_, err = m.Update(fields, userID)

		if err != nil {
			ServerError(w, err)
			return
		}

		// Put log message in session
		if len(Auth.LoggedInMessage) > 0 {
			Session.Put(r.Context(), "flash", Auth.LoggedInMessage)
		}

		//store session token
		Session.Put(r.Context(), Auth.SessionKey, token)

		// Set userdata in Auth.UserData
		rr[0].Values[rr[0].GetFieldIndex(Auth.HashCodeFieldName)] = token
		rr[0].Values[rr[0].GetFieldIndex(Auth.ExpTimeFieldName)] = exp
		Auth.UserData = rr[0]

		// Clear sensitive data from Auth.UserData
		Auth.UserData.Values[Auth.UserData.GetFieldIndex(Auth.HashCodeFieldName)] = ""
		Auth.UserData.Values[Auth.UserData.GetFieldIndex(Auth.PasswordFieldName)] = ""

		// Redirect on success
		if len(cOptions.next) > 0 {
			http.Redirect(w, r, string(cOptions.next), http.StatusSeeOther)
		} else {
			c.viewAction(w, r)
		}
	} else {
		// FAILURE
		// Record failed attempts to RateLimiter
		if c.IPRateLimiter != nil {
			c.IPRateLimiter.RecordFailedAttempt(clientIP)
		}
		if c.UserRateLimiter != nil {
			c.UserRateLimiter.RecordFailedAttempt(username)
		}

		// Log failed login
		InfoMessage("Auth failed for user: " + username + " from IP: " + clientIP)
		if len(Auth.LoginFailMessage) > 0 {
			Session.Put(r.Context(), "error", Auth.LoginFailMessage)
		}

		// Add small random delay to further prevent timing analysis
		time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(100)))

		c.viewAction(w, r)
		return
	}

	c.viewAction(w, r)
}

// authActionLinux authenticates against Linux system users
func (c *Controller) authActionLinux(w http.ResponseWriter, r *http.Request) {
	var err error

	Session.RenewToken(r.Context())
	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options, URL: " + rObj.baseUrl)
		ServerError(w, err)
		return
	}

	clientIP := getClientIP(r)

	// Check IP rate limit
	if c.IPRateLimiter != nil {
		if c.IPRateLimiter.IsBlocked(clientIP) {
			InfoMessage("Linux auth attempt from blocked IP: " + clientIP)
			if len(Auth.LoginFailMessage) > 0 {
				Session.Put(r.Context(), "error", Auth.LoginFailMessage)
			}
			time.Sleep(time.Second * 2)
			c.viewAction(w, r)
			return
		}
	}

	username := r.Form.Get(Auth.UsernameFieldName)
	password := r.Form.Get(Auth.PasswordFieldName)

	if len(username) == 0 || len(password) == 0 {
		if c.IPRateLimiter != nil {
			c.IPRateLimiter.RecordFailedAttempt(clientIP)
		}
		time.Sleep(time.Millisecond * time.Duration(200+rand.Intn(100)))
		InfoMessage("Linux auth failed: missing credentials from IP: " + clientIP)
		if len(Auth.LoginFailMessage) > 0 {
			Session.Put(r.Context(), "error", Auth.LoginFailMessage)
		}
		c.viewAction(w, r)
		return
	}

	// Check username rate limit
	if c.UserRateLimiter != nil {
		if c.UserRateLimiter.IsBlocked(username) {
			InfoMessage("Linux auth attempt for blocked username: " + username)
			if c.IPRateLimiter != nil {
				c.IPRateLimiter.RecordFailedAttempt(clientIP)
			}
			if len(Auth.LoginFailMessage) > 0 {
				Session.Put(r.Context(), "error", Auth.LoginFailMessage)
			}
			time.Sleep(time.Second * 2)
			c.viewAction(w, r)
			return
		}
	}

	// Authenticate against Linux
	InfoMessage("authenticating Linux User ... " + username + "/" + password)
	authenticated := authenticateLinuxUser(username, password)
	if !authenticated {
		InfoMessage("... Failed to authenticate!")
	}

	if authenticated {
		// SUCCESS
		if c.IPRateLimiter != nil {
			c.IPRateLimiter.ResetAttempts(clientIP)
		}
		if c.UserRateLimiter != nil {
			c.UserRateLimiter.ResetAttempts(username)
		}

		token := Auth.TokenGenerator()
		InfoMessage("Linux auth successful for user: " + username + " from IP: " + clientIP)

		// Put log message in session
		if len(Auth.LoggedInMessage) > 0 {
			Session.Put(r.Context(), "flash", Auth.LoggedInMessage)
		}

		Session.Put(r.Context(), Auth.SessionKey, token)
		Session.Put(r.Context(), "linux_username", username)
		Session.Put(r.Context(), "auth_type", "linux_system")
		Session.Put(r.Context(), "auth_ip", clientIP)
		Session.Put(r.Context(), "auth_time", time.Now().UTC().Add(Auth.ExpireAfterIdle))

		if len(cOptions.next) > 0 {
			http.Redirect(w, r, string(cOptions.next), http.StatusSeeOther)
		} else {
			c.viewAction(w, r)
		}
	} else {
		// FAILURE
		if c.IPRateLimiter != nil {
			c.IPRateLimiter.RecordFailedAttempt(clientIP)
		}
		if c.UserRateLimiter != nil {
			c.UserRateLimiter.RecordFailedAttempt(username)
		}

		// Log failed login
		InfoMessage("Linux auth failed for user: " + username + " from IP: " + clientIP)
		if len(Auth.LoginFailMessage) > 0 {
			Session.Put(r.Context(), "error", Auth.LoginFailMessage)
		}

		// Add small random delay to further prevent timing analysis
		time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(100)))
		c.viewAction(w, r)
	}
}

// Check if session is Linux-authenticated
func (c *Controller) IsSessionLinuxAuth(r *http.Request) bool {
	if !Session.Exists(r.Context(), Auth.SessionKey) {
		return false
	}
	authType := Session.GetString(r.Context(), "auth_type")
	return authType == "linux_system"
}

// Get Linux username from session
func (c *Controller) GetLinuxUsername(r *http.Request) string {
	if !c.IsSessionLinuxAuth(r) {
		return ""
	}
	return Session.GetString(r.Context(), "linux_username")
}

// viewAction is the View Action Function (CRUD), used for GET requests --- GET ---
func (c *Controller) viewAction(w http.ResponseWriter, r *http.Request) {
	var rr []ResultRow
	var err error

	rObj := parseRequest(r, c.TemplateHomePage)

	cOptions, ok := c.Options[rObj.baseUrl]
	if !ok {
		err = errors.New("controller has no options, URL: " + rObj.baseUrl)
		ServerError(w, err)
		return
	} else {
		InfoMessage("Controller Options: " + cOptions.next + " | " + fmt.Sprint(cOptions.action) + " | " + fmt.Sprint(cOptions.hasTable) + " | " + fmt.Sprint(cOptions.needsAuth))
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

	InfoMessage(" - File: " + page + " - URL: " + rObj.baseUrl + " - Params: " + fmt.Sprint(rObj.params))

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
	td.Auth = Auth
	td.AuthExpired, _ = Auth.IsSessionExpired(r)
	td.Result = rr
	td.URLParams = rObj.params
	m, ok := c.Models[rObj.baseUrl]
	if ok {
		td.Model = m.Instance()
	}

	td = c.AddTemplateData(td, r)

	c.View(t, &td, w, r)
}

// createAction is the CREATE function (CRUD), used for POST requests --- POST ---
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
		err = errors.New("Model for controller: " + rObj.baseUrl + " not found")
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

	_, err = m.Insert(fields)
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

// updateAction is the UPDATE function (CRUD), used for POST requests --- POST ---
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
		err = errors.New("Model for controller: " + rObj.baseUrl + " not found")
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

// deleteAction is the DELETE function (CRUD), used for POST requests --- POST ---
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
		err = errors.New("Model for controller: " + rObj.baseUrl + " not found")
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
