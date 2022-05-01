# MySQL - MVC (Model View Controller) with Golang

MVC (Model View Controller) implementation with Golang using MySql database

## Overview
This is a Golang package easy to use and build almost any MVC Web App connected to MySql database with just a few steps.</br>
`gomvc` package requires a MySql Server up and running and a database ready to drive your web application.</br>

Build a standard MVC (Model, View, Controller) style web app with minimum Golang code, like you use a classic MVC Framework.</br>
Many features, many ready to use functions, highly customizable, embeded log and error handling 

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

* Easy to learn, use and build.
* Flexibility and Customization
* Embeded Authentication using database table if needed. 
* Embeded libraries like : session managment, csrf middleware, http router
* Embeded info log and server error handling
* Strong MVC pattern
* Models with many features and easy to use functions
* Models with build in relational functionlity with other database tables
* Controlles with simple yet powerful http handling
* Ability to attach our own functions to Controller and requests for more customized http handling
* Working Examples to cover almost every case.

## Installation

This package requires Go 1.12 or newer.

```
$ go get github.com/kostasdak/gomvc
```

Note: If you're using the traditional `GOPATH` mechanism to manage dependencies, instead of modules, you'll need to `go get` and `import` `github.com/kostasdak/gomvc`

</br>
</br>

## Template file names general rules
It is recomended to use the folowing rule for template filenames.
All template files must have `.tmpl` extension

* Template layout</br>
This file is the layout of your app `base.layout.html`

* Pages</br>
If page needs a data from a databese table use this pattern : `[dbtable].[action].tmpl`

```
products view page : products.view.tmpl
products create page : products.create.tmpl
products edit page : products.edit.tmpl
products delete page : products.delete.tmpl
```

Routes can use the same file depending how the template code is writen, for example :
products create & edit page : `products.edit.tmpl (template has code to handle both)`

Pages without data connection / static pages : [pagename].[action].tmpl

page about : about.view.tmpl
page contact : contact.view.tmpl

* Home Page
Same rule like all the above pages 

page home : home.view.tmpl

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
action : [</br>
`gomvc.ActionView` View (GET) /</br>
`gomvc.ActionCreate` Create (POST) /</br>
`gomvc.ActionUpdate` Update (POST) /</br>
`gomvc.ActionDelete` Delete (POST)</br>
]</br>
model  : database `gomvc.model` object</br>
</br>

```
func AppHandler(db *sql.DB, cfg *gomvc.AppConfig) http.Handler {

	// initialize controller
	c.Initialize(db, cfg)

	// load template files ... path : /web/templates
	// required : homepagefile & template file
	// see [template names] for details
	c.CreateTemplateCache("home.view.tmpl", "base.layout.html")

	// *** Start registering urls, actions and models ***

	// RegisterAction(url, next, action, model)
	// url = url routing path
	// next = redirect after action complete, use in POST actions if necessary
	// model = database model object for CRUD operations

	// home page : can have two urls "/" and "/home"
	c.RegisterAction("/", "", gomvc.ActionView, nil)
	c.RegisterAction("/home", "", gomvc.ActionView, nil)

	// create model for [products] database table
	// use the same model for all action in this example
	pModel := gomvc.Model{DB: db, PKField: "id", TableName: "products"}

	// view products ... / show all products || /products/view/{id} for one product
	c.RegisterAction("/products", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/view/*", "", gomvc.ActionView, &pModel)

	// build create product action ... this url has two actions
	// #1 View page -> empty product form no redirect url (no next url required)
	// #2 Post form data to create a new record in table [products] -> then redirect to [next] url -> products page
	c.RegisterAction("/products/create", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/create", "/products", gomvc.ActionCreate, &pModel)

	// build edit product actions ... this url has two actions
	// #1 View page with the product form -> edit form (no next url required)
	// #2 Post form data to update record in table [products] -> then redirect to [next] url -> products page
	c.RegisterAction("/products/edit/*", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/edit/*", "/products", gomvc.ActionUpdate, &pModel)

	// build delete product actions ... this url has two actions
	// #1 View page with the product form -> edit form [locked] to confirm detetion (no next url required)
	// #2 Post form data to delete record in table [products] -> then redirect to [next] url -> products page
	c.RegisterAction("/products/delete/*", "", gomvc.ActionView, &pModel)
	c.RegisterAction("/products/delete/*", "/products", gomvc.ActionDelete, &pModel)

	// build about page ... static page, no table/model, no [next] url
	c.RegisterAction("/about", "", gomvc.ActionView, nil)

	// build contact page ... static page, no table/model, no [next] url
	c.RegisterAction("/contact", "", gomvc.ActionView, nil)

	// build contact page POST action ... static page, no table/model, no [next] url
	// Demostrating how to register a custom func to handle the http request/response using your oun code
	// and handle POST data and have access to database through the controller and model object
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

	//test : I have access to products model !!!
	fmt.Print("\n\n")
	fmt.Println("********** ContactPostForm **********")
	fmt.Println("Table Fields : ", c.Models["/products"].Fields)

	//read data from table products (Model->products) even if this is a POST action for contact page
	fmt.Print("\n\n")
	rows, _ := c.Models["/products"].GetRecords([]gomvc.Filter{}, 100)
	fmt.Println("Select Rows Example 1 : ", rows)

	//read data from table products (Model->products) even if this is a POST action for contact page
	fmt.Print("\n\n")
	id, _ := c.Models["/products"].GetLastId()
	fmt.Println("Select Rows Example 1 : ", id)

	//read data from table products (Model->products) with filter (id=1)
	fmt.Print("\n\n")
	var f = make([]gomvc.Filter, 0)
	f = append(f, gomvc.Filter{Field: "id", Operator: "=", Value: 1})
	rows, err := c.Models["/products"].GetRecords(f, 0)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Select Rows Example 2 : ", rows)

	//test : Print Posted Form fields
	fmt.Print("\n\n")
	fmt.Println("Form fields : ", r.Form)

	//test : Set session message
	c.GetSession().Put(r.Context(), "error", "Hello From Session")

	//redirect to homepage
	http.Redirect(w, r, "/", http.StatusSeeOther)

}
```

## More Examples ...

[Example 01](https://github.com/kostasdak/go-mvc-example-1) - basic use of gomvc, one table [products]

[Example 02](https://github.com/kostasdak/go-mvc-example-2) - basic use of gomvc, two tables related [products]->[colors] (one-to-many relation)

[Example 03](https://github.com/kostasdak/go-mvc-example-3) - gomvc Auth example, two tables related [products]->[colors] (one-to-many relation)


TO DO : more examples