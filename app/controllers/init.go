package controllers

import "github.com/robfig/revel"
import "database/sql"
import _ "github.com/herenow/go-crate"
import "log"

var Db *sql.DB

func dbConnect() {
	// Configurations
	driver, err := revel.Config.String("db.driver")

	if err == false {
		log.Fatal("Missing DB driver conf")
	}

	spec, err := revel.Config.String("db.endpoint")

	if err == false {
		log.Fatal("Missing DB endpoint conf")
	}

	log.Println("Connecting to DB %s using Driver %s", spec, driver)

	// Connect
	Db, _ = sql.Open(driver, spec)
}

func init() {
	revel.OnAppStart(dbConnect)
}
