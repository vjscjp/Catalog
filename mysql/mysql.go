package mysql

import (
	"database/sql"
	"fmt"
	"os"

	log "github.com/CiscoCloud/shipped-common/logging"
	_ "github.com/go-sql-driver/mysql"
)


var
// Create the catalog database
func create_db () {

}

// Get environment variable.  Return error if not set.
func getenv(name string) (val string, e error) {
	val = os.Getenv(name)
	if val == "" {
		s := "Required environment variable not found: %s"
		log.Error.Printf(s, name)
		return "", fmt.Errorf(s, name)
	}
	return val, nil
}

func init_db() (db *sql.DB, e error) {

	db_host, e := getenv("SHIPPED_MYSQL_HOST")
	if e != nil {
		return nil, e
	}
	db_schema, e := getenv("SHIPPED_MYSQL_SCHEMA")
	if e != nil {
		return nil, e
	}
	db_user, e := getenv("SHIPPED_MYSQL_USER")
	if e != nil {
		return nil, e
	}
	db_password, e := getenv("SHIPPED_MYSQL_PASSWORD")
	if e != nil {
		return nil, e
	}

	cxn := fmt.Sprintf("%s:%s@%s/%s", db_user, db_password, db_host, db_schema)
	db, e = sql.Open("mysql", cxn)
	if e != nil {
		log.Error.Printf("error getting db object: %s", e.Error())
		return nil, e
	}
	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = db.Ping()
	if e != nil {
		log.Error.Printf("Error from db.Ping: %s", e.Error())
		return nil, e
	}
	return db, nil
}

// Read from one of the preinstalled tables
func mysql() (e error) {
	log.Info.Printf("Testing go and mysql")

	db, err := init_db()
	if err != nil {
		log.Error.Printf("Error from init_db: %s", err.Error())
	}

	var (
		id         int
		first_name string
		last_name  string
	)
	rows, err := db.Query("select actor_id, first_name, last_name from actor")
	if err != nil {
		log.Error.Printf("Error from db.Query: %s", err.Error())
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &first_name, &last_name)
		if err != nil {
			log.Error.Printf("Error from rows.Scan: %s", err.Error())
			return err
		}
		log.Info.Printf("%d  %s  %s", id, first_name, last_name)
	}
	err = rows.Err()
	if err != nil {
		log.Error.Printf("Error from rows: %s", err.Error())
		return err
	}

	return nil

}
