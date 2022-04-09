# MySQL - MVC (Model View Controller) with Golang

MVC (Model View Controller) implementation with Golang using MySql database

## Overview
This is a Golang package easy to use and build almost any MVC Web App connected to MySql database with just a few steps.</br>
`gomvc` package requires a MySql Server up and running and a database ready to drive your web application.</br>

Build a standard MVC (Model, View, Controller) style web app with minimum Golang code, like you use a classic MVC Framework. </br>
</br>

#### MVC

```
(databse CRUD)      (http req/resp) 
     Model <--------> Controller
         \            /    
          \          /
           \        /
            \      /
             \    /
              View
      (text/template files)
```

#### Basic Steps
* Edit the config file
* Load config file `config.yaml`
* Connect to MySql database 
* Write code to initialize your Models and Controllers
* Write your standard text/Template files (Views)
* Start your server and enjoy


#### This package includes :</br>
MySql Driver : `github.com/go-sql-driver/mysql`</br>
http Router : `github.com/go-chi/chi/v5`</br>
csrf middleware :`github.com/justinas/nosurf`</br>
Session middleware : `github.com/alexedwards/scs/v2`</br>
Config Loader :`github.com/spf13/viper`</br>
</br>

## Features










## Installation

This package requires Go 1.12 or newer.

```
$ go get github.com/kostasdak/gomvc
```

Note: If you're using the traditional `GOPATH` mechanism to manage dependencies, instead of modules, you'll need to `go get` and `import` `github.com/kostasdak/gomvc`

</br>
</br>

## Basic Use

* Edit configuration file, 

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

### `func main()`

Create controller variable in your `main.go` file outside the `func main()`
Controller must be accessible from all functions in main package

`var c gomvc.Controller`

* Load Configuration file 

`cfg := gomvc.LoadConfig("./configs/config.yml")`
	
* Connect to database

```
db, err := gomvc.ConnectDatabase(cfg.Database.Dbuser, cfg.Database.Dbpass, cfg.Database.Dbname)
if err != nil {
	log.Fatal(err)
	return
}
defer db.Close()
```

* Start web server

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

#### main()
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

#### AppHandler()

* initialize the controller
* load your template files into cache
* initialize your models
* register urls and actions 

`RegisterAction (route, next, action, model)`</br>
route  : `string` "url"</br>
next   : `string` "url" (used after POST to redirect browser)</br>
action : [ `gomvc.ActionView` View (GET) / `gomvc.ActionCreate` Create (POST) / `gomvc.ActionUpdate` Update (POST) / `gomvc.ActionDelete` Delete (POST) ]</br>
model  : database `gomvc.model` object</br>
</br>

```
func AppHandler(db *sql.DB, cfg *gomvc.AppConfig) http.Handler {

	// initialize
	c.Initialize(db, cfg)

	// load template files ... path : /web/templates
	c.CreateTemplateCache("home.view.tmpl", "base.layout.html")

	// *** Start registering urls, actions and models ***
	// home page
	c.RegisterAction("/", "", gomvc.ActionView, nil)
	c.RegisterAction("/home", "", gomvc.ActionView, nil)

	// create model for [products] table
	pModel := gomvc.Model{DB: db, IdField: "id", TableName: "products"}

	// view products ... /products for all records || /products/view/{id} for one product
	c.RegisterAction("/products", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/view/*", "", gomvc.ActionView, &pModel)

	// create product actions ... this url has two actions
	// #1 View page -> empty form (no next url required)
	// #2 Post form data to create a new record in table [products] -> then redirect to [next] url
	c.RegisterAction("/products/create", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/create", "/products", gomvc.ActionCreate, &pModel)

	// create edit product actions ... this url has two actions
	// #1 View page with product data -> edit form (no next url required)
	// #2 Post form data to update record in table [products] -> then redirect to [next] url
	c.RegisterAction("/products/edit/*", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/edit/*", "/products", gomvc.ActionUpdate, &pModel)

	// create delete product actions ... this url has two actions
	// #1 View page with product data -> edit form [locked] to confirm detetion (no next url required)
	// #2 Post form data to delete record in table [products] -> then redirect to [next] url
	c.RegisterAction("/products/delete/*", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/delete/*", "/products", gomvc.ActionDelete, &pModel)

	// create about page ... static page, no table/model, no [next] url
	c.RegisterAction("/about", "", gomvc.ActionView, nil)

	// contact page ... static page, no table/model, no [next] url
	c.RegisterAction("/contact", "", gomvc.ActionView, nil)

	// contact page POST action ... static page, no table/model, no [next] url
	// Register a custom func to handle the request/response using your oun code
	c.RegisterCustomAction("/contact", "", gomvc.HttpPOST, nil, ContactPostForm)
	return c.Router
}
```

#### Custom Action func ContactPostForm()
Build a custom func to handle a specific action or url.
This example handles the POST request from a contact form.

```
// Custom handler for specific page and action
func ContactPostForm(w http.ResponseWriter, r *http.Request) {

	//test ... I have access to products model !!!
	fmt.Print("Table Fields : ")
	fmt.Println(c.Models["products"].Fields)

	//read data from table products even this is a POST action for contact page
	rows, _ := c.Models["products"].GetAllRecords(100)
	fmt.Print("Select Rows : ")
	fmt.Println(rows)

	//test form fields
	fmt.Print("Form Fields : ")
	fmt.Println(r.Form)

	//test session
	c.GetSession().Put(r.Context(), "error", "Hello From Session")

	//redirect to homepage
	http.Redirect(w, r, "/", http.StatusSeeOther)

}
```

## More Examples ...

[Example 01](https://github.com/kostasdak/go-mvc-example-1) - basic use of gomvc, one table [products]

[Example 02](https://github.com/kostasdak/go-mvc-example-2) - basic use of gomvc, two related tables [products]->[colors] (one-to-many relation)

TO DO : more links, more examples