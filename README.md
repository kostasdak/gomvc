# MVC (Model View Controller) with Golang

MVC (Model View Controller) implementation with Golang and MySql database

## Overview
This project is a starting point to create a Golang package that can be used to build any MVC Web App connected with MySql database with just a few easy steps.


## Installation

This package requires Go 1.12 or newer.

```
$ go get github.com/kostasdak/gomvc
```

Note: If you're using the traditional `GOPATH` mechanism to manage dependencies, instead of modules, you'll need to `go get` and `import` `github.com/kostasdak/gomvc`


### Basic Use
  
In your main.go file create a controller variable

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