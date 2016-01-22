package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	log "github.com/CiscoCloud/shipped-common/logging"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB // Database object designed to be long lived

type CatalogItem struct {
	Image       string  `json:"image"`
	ItemID      int     `json:"item_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type CatalogItems struct {
	Items []CatalogItem `json:"items"`
}

type dbCreds struct {
	db_schema   string
	db_user     string
	db_password string
	db_host     string
}

func main() {

	e := createDatabase()
	if e != nil {
		log.Error.Printf("Error creating database: %s", e.Error())
		os.Exit(1)
	}

	db, e = getDBObject()
	if e != nil {
		log.Error.Printf("Error getting db object: %s", e.Error())
		os.Exit(1)
	}

	http.HandleFunc("/v1/catalog/", Catalog)
	http.HandleFunc("/", HandleIndex)

	http.ListenAndServe(":8000", nil)
}

// HandleIndex this is the index endpoint will return 200
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	lp := path.Join("templates", "layout.html")
	fp := path.Join("templates", "index.html")

	// Note that the layout file must be the first parameter in ParseFiles
	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func getdbCreds() (creds dbCreds, e error) {
	var x dbCreds

	x.db_host, e = getenv("SHIPPED_MYSQL_HOST")
	if e != nil {
		return creds, e
	}
	x.db_schema, e = getenv("SHIPPED_MYSQL_SCHEMA")
	if e != nil {
		return creds, e
	}
	x.db_user, e = getenv("SHIPPED_MYSQL_USER")
	if e != nil {
		return creds, e
	}
	x.db_password, e = getenv("SHIPPED_MYSQL_PASSWORD")
	if e != nil {
		return creds, e
	}
	return x, nil
}

// Get a (long lived) database object
func getDBObject() (db *sql.DB, e error) {
	creds, e := getdbCreds()
	if e != nil {
		log.Error.Printf("Error getting database creds: %s", e.Error())
		return
	}

	// Create the db object
	cxn := fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, creds.db_schema)
	dbx, e := sql.Open("mysql", cxn)
	if e != nil {
		log.Error.Printf("error getting db object: %s", e.Error())
		return
	}

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Error.Printf("Error from db.Ping: %s", e.Error())
		return
	}
	return dbx, nil
}

// Create the shipped database if it does not exist
// then populate the catalog table from the json defined rows
func createDatabase() (e error) {
	var create_database string = `
	CREATE DATABASE IF NOT EXISTS %s`

	var create_table string = `
	CREATE TABLE IF NOT EXISTS catalog
	(
	item_id INT PRIMARY KEY,
	name    VARCHAR(255) NOT NULL,
	description VARCHAR(255) NOT NULL,
	price   FLOAT NOT NULL,
	image   VARCHAR(255)
	)`

	var insert_table string = `
	INSERT IGNORE INTO catalog (item_id, name, description, price, image) VALUES (?,?,?,?,?)`

	creds, e := getdbCreds()
	if e != nil {
		return e
	}

	// Initially the shipped catalog db may not exist,
	// so connect with a database that will always exist.
	cxn := fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, "information_schema")
	dbx, e := sql.Open("mysql", cxn)
	if e != nil {
		log.Error.Printf("error getting db object: %s", e.Error())
		return e
	}
	defer dbx.Close()

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Error.Printf("Error from db.Ping: %s", e.Error())
		return e
	}

	// Create the database
	_, e = dbx.Exec(fmt.Sprintf(create_database, creds.db_schema))
	if e != nil {
		log.Error.Printf("Error creating database: %s", e.Error())
		os.Exit(1)
	}

	// Close the old db object and get new one for shipped database.
	dbx.Close()
	cxn = fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, creds.db_schema)
	dbx, e = sql.Open("mysql", cxn)
	if e != nil {
		log.Error.Printf("error getting db object: %s", e.Error())
		return e
	}
	defer dbx.Close()

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Error.Printf("Error from db.Ping: %s", e.Error())
		return e
	}

	// Create the catalog table
	_, e = dbx.Exec(create_table)
	if e != nil {
		log.Error.Printf("Error creating database: %s", e.Error())
		return e
	}

	// Get database rows defined as json
	var cis CatalogItems
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		log.Error.Printf("Error reading catalog json file: %s", e.Error())
		return e
	}
	json.Unmarshal(file, &cis)

	// Get a database transaction (ensures all operations use same connection)
	tx, e := dbx.Begin()
	if e != nil {
		log.Error.Printf("Error getting database transaction: %s", e.Error())
		return e
	}
	defer tx.Rollback()

	// Prepare insert command
	stmt, e := tx.Prepare(insert_table)
	if e != nil {
		log.Error.Printf("Error creating prepared statement: %s", e.Error())
		return e
	}
	defer stmt.Close()

	// Populate rows of catalog table
	for _, item := range cis.Items {
		_, e = stmt.Exec(item.ItemID, item.Name, item.Description, item.Price, item.Image)
		if e != nil {
			log.Error.Printf("Error inserting row into catalog table: %s", e.Error())
			return e
		}
	}

	e = tx.Commit()
	if e != nil {
		log.Error.Printf("Error during transaction commit: %s", e.Error())
		return e
	}

	stmt.Close()
	dbx.Close()

	return nil
}

// Get a single catalog row
func getCatalogItem(item int) (ci CatalogItem, e error) {
	//var ci CatalogItem
	e = db.QueryRow("SELECT item_id, name, description, price, image FROM catalog WHERE item_id = ?", item).Scan(
		&ci.ItemID, &ci.Name, &ci.Description, &ci.Price, &ci.Image)
	if e != nil {
		log.Error.Printf("Error reading database row for item %d: %s", item, e.Error())
		return ci, e
	}
	return ci, nil
}

// Get the whole catalog from the database
func getCatalog() (cat CatalogItems, e error) {

	rows, e := db.Query("SELECT  item_id, name, description, price, image FROM catalog")
	if e != nil {
		log.Error.Printf("Error from db.Query: %s", e.Error())
		return
	}
	defer rows.Close()

	var ci CatalogItem
	var cis CatalogItems

	for rows.Next() {
		err := rows.Scan(&ci.ItemID, &ci.Name, &ci.Description, &ci.Price, &ci.Image)
		if err != nil {
			log.Error.Printf("Error from rows.Scan: %s", err.Error())
			return
		}
		cis.Items = append(cis.Items, ci)
	}
	e = rows.Err()
	if e != nil {
		log.Error.Printf("Error from rows: %s", e.Error())
		return
	}
	rows.Close()

	return cis, nil
}

// Catalog:  this will return an item or the whole list
func Catalog(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		// Check Parameter
		uriSegments := strings.Split(r.URL.Path, "/")
		var itemNumber = 0
		if len(uriSegments) >= 3 {
			itemNumber, _ = strconv.Atoi(uriSegments[3])
		}
		if itemNumber > 0 {
			// Send catalog by item_id
			ci, e := getCatalogItem(itemNumber)
			if e != nil {
				response := fmt.Sprintf("Error from database retrieving item_id %d: %s", itemNumber, e.Error())
				w.Write([]byte(response))
				return
			}
			response, err := json.MarshalIndent(ci, "", "    ")
			if err != nil {
				log.Error.Printf("Error marshalling returned catalog item %s", err.Error())
				return
			}
			w.Write([]byte(response))
			log.Info.Printf("Succesfully sent item_number: %d", itemNumber)
		} else {
			// Send full catalog
			cis, e := getCatalog()
			if e != nil {
				log.Error.Printf("Error getting catalog items: %s", e.Error())
				return
			}
			response, err := json.MarshalIndent(cis.Items, "", "    ")
			if err != nil {
				log.Error.Printf("Error marshalling returned catalog item %s", err.Error())
				return
			}
			w.Write([]byte(response))
			log.Info.Printf("Succesfully sent %d catalog items", len(cis.Items))
		}
	case "POST":
		// Do we want to create records?
	case "PUT":
		// Update an existing record.
	case "DELETE":
		// Remove the record.
	default:
		// Give an error message.
		w.Write([]byte(fmt.Sprintf("Error: Unsupported HTTP method: %s", r.Method)))
	}
}
