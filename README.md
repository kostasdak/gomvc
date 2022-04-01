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

In your `main.go` file create a controller variable

`var c gomvc.Controller`

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
