# MVC (Model View Controller) with Golang

MVC (Model View Controller) implementation with Golang using MySql database

## Overview
This is a Golang package that can be used to build any MVC Web App connected to MySql database with just a few easy steps.
`gomvc` package requires MySql Server up and running and a database ready to drive your web app.


## Installation

This package requires Go 1.12 or newer.

```
$ go get github.com/kostasdak/gomvc
```

Note: If you're using the traditional `GOPATH` mechanism to manage dependencies, instead of modules, you'll need to `go get` and `import` `github.com/kostasdak/gomvc`


### Basic Use

Edit configuration file, 

```
#UseCache true/false 
#Read files for every request, use this option for debug and development, set to true on production server
UseCache: false

#EnableInfoLog true/false
#Enable information log in console window, set to false in production server
EnableInfoLog: true

#InfoFile "path.to.filename"
#Set info filename, direct info log to file instead of console window
InfoFile: ""

#ShowStackOnError true/false
#Set to true to see the stack error trace in web page error report, set to false in production server
ShowStackOnError: true

#ErrorFile "path.to.filename"
#Set error filename, direct error log to file instead of web page, set this file name in production server
ErrorFile: ""

#Server Settings
server:
  #Listening port
  port: 8080

  #Session timeout in hours 
  sessionTimeout: 24

  #Use secure session, set to tru in production server
  sessionSecure: true

#Database settings
database:
  #Database name
  dbname: "golang"

  #Database server/ip address
  server: "localhost"

  #Database user
  dbuser: "root"

  #Database password
  dbpass: ""
```

Create controller variable in your `main.go` file outside the `func main()`
Controller must be accessible from all functions in main package

`var c gomvc.Controller`

### `func main()`

Load Configuration file 

`cfg := gomvc.LoadConfig("./configs/config.yml")`
	
Connect to database

```
db, err := gomvc.ConnectDatabase(cfg.Database.Dbuser, cfg.Database.Dbpass, cfg.Database.Dbname)
if err != nil {
	log.Fatal(err)
	return
}
defer db.Close()
```

Start your server

```
srv := &http.Server{
	Addr:    ":" + strconv.FormatInt(int64(cfg.Server.Port), 10),
	Handler: AppHandler(db, cfg),
}

fmt.Println("Web app starting at port : ", cfg.Server.Port)

err = srv.ListenAndServe()
if err != nil {
	log.Fatal(err)
}
```

#### func main()
```
func main() {

	// Load Configuration file
	cfg := gomvc.LoadConfig("./config/config.yml")

	// Connect to database
	db, err := gomvc.ConnectDatabase(cfg.Database.Dbuser, cfg.Database.Dbpass, cfg.Database.Dbname)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	//Start Server
	srv := &http.Server{
		Addr:    ":" + strconv.FormatInt(int64(cfg.Server.Port), 10),
		Handler: AppHandler(db, cfg),
	}

	fmt.Println("Web app starting at port : ", cfg.Server.Port)

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
```

#### AppHandler

* initialize the controller
* load your template files into cache
* register your actions ... view, create, edit, delete
* boom your web app works !!!

```
func AppHandler(db *sql.DB, cfg *gomvc.AppConfig) http.Handler {

	// initialize
	c.Initialize(db, cfg)
	c.CreateTemplateCache("home.view.tmpl", "base.layout.html")

	// home page
	c.RegisterAction("/", "", gomvc.ActionView, "")
	c.RegisterAction("/home", "", gomvc.ActionView, "")

	// view products
	c.RegisterAction("/products", "", gomvc.ActionView, "products")
	c.RegisterAction("/products/view/*", "", gomvc.ActionView, "products")

	// create product
	c.RegisterAction("/products/create", "", gomvc.ActionView, "products")
	c.RegisterAction("/products/create", "/products", gomvc.ActionCreate, "products")

	// edit product
	c.RegisterAction("/products/edit/*", "", gomvc.ActionView, "products")
	c.RegisterAction("/products/edit/*", "/products", gomvc.ActionUpdate, "products")

	// delete product
	c.RegisterAction("/products/delete/*", "", gomvc.ActionView, "products")
	c.RegisterAction("/products/delete/*", "/products", gomvc.ActionDelete, "products")

	// about page
	c.RegisterAction("/about", "", gomvc.ActionView, "")

	// contact page
	c.RegisterAction("/contact", "", gomvc.ActionView, "")

	return c.Router
}
```
