package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/kostasdak/gomvc"
)

var c gomvc.Controller

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

// App handler ... Builds the structure of the app !!!
func AppHandler(db *sql.DB, cfg *gomvc.AppConfig) http.Handler {

	// initialize
	c.Initialize(db, cfg)
	c.CreateTemplateCache("home.view.tmpl", "base.layout.html")

	// home page
	c.RegisterAction("/", "", gomvc.ActionView, nil)
	c.RegisterAction("/home", "", gomvc.ActionView, nil)

	pViewModel := gomvc.Model{DB: db, IdField: "id", TableName: "products"}

	// view products
	c.RegisterAction("/products", "", gomvc.ActionView, &pViewModel)
	c.RegisterAction("/products/view/*", "", gomvc.ActionView, &pViewModel)

	// create product
	c.RegisterAction("/products/create", "", gomvc.ActionView, &pViewModel)
	c.RegisterAction("/products/create", "/products", gomvc.ActionCreate, &pViewModel)

	// edit product
	c.RegisterAction("/products/edit/*", "", gomvc.ActionView, &pViewModel)
	c.RegisterAction("/products/edit/*", "/products", gomvc.ActionUpdate, &pViewModel)

	// delete product
	c.RegisterAction("/products/delete/*", "", gomvc.ActionView, &pViewModel)
	c.RegisterAction("/products/delete/*", "/products", gomvc.ActionDelete, &pViewModel)

	// about page
	c.RegisterAction("/about", "", gomvc.ActionView, nil)

	// contact page
	c.RegisterAction("/contact", "", gomvc.ActionView, nil)

	c.RegisterCustomAction("/contact", "", gomvc.HttpPOST, nil, ContactPostForm)
	return c.Router
}

// Custom handler for specific page and action
func ContactPostForm(w http.ResponseWriter, r *http.Request) {
	//test if I have access to products
	fmt.Print("Table Fields : ")
	fmt.Println(c.Models["products"].Fields)

	rows, _ := c.Models["products"].GetAllRecords(100)
	fmt.Print("Select Rows : ")
	fmt.Println(rows)

	//test form
	fmt.Print("Form Fields : ")
	fmt.Println(r.Form)
	//for k, v := range r.Form {
	//	fmt.Println(k, v)
	//}

	//test session
	c.GetSession().Put(r.Context(), "error", "Hello From Session")

	//TO DO : send email
	//test email ... failed
	/*from := "kostas@domain.com"
	auth := smtp.PlainAuth("Kostas", from, "*******", "mail.domain.com")
	err = smtp.SendMail("mail.domain.com:25", auth, from, []string{"kostas@domain.com"}, []byte("Hello, world"))
	if err != nil {
		fmt.Println(err)
	}*/

	//redirect to homepage
	http.Redirect(w, r, "/", http.StatusSeeOther)

}
