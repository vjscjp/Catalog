package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB // Database object designed to be long lived

type CatalogType struct {
	Items []struct {
		Image       string  `json:"image"`
		ItemID      int     `json:"item_id"`
		Name        string  `json:"name"`
		Descrpition string  `json:"description"`
		Price       float64 `json:"price"`
	} `json:"items"`
}

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

type Response struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const errors = "ERROR"
const success = "SUCCESS"

func main() {

	e := createDatabase()
	if e != nil {
		log.Printf("Error creating database: %s", e.Error())
	}

	db, e = getDBObject()
	if e != nil {
		log.Printf("Error getting db object: %s", e.Error())
	}

	http.HandleFunc("/v1/catalog/", Catalog)
	http.HandleFunc("/", HandleIndex)

	http.ListenAndServe(":8000", nil)
}

// Get environment variable.  Return error if not set.
func getenv(name string, dflt string) (val string, e error) {
	val = os.Getenv(name)
	if val == "" {
		val = dflt
		if val == "" {
			s := "Required environment variable not found: %s"
			log.Printf(s, name)
			return "", fmt.Errorf(s, name)
		}
	}
	return val, nil
}

func getdbCreds() (creds dbCreds, e error) {
	var x dbCreds

	x.db_host, e = getenv("SHIPPED_MYSQL_HOST", "tcp(mysql:3306)")
	if e != nil {
		return creds, e
	}
	x.db_schema, e = getenv("SHIPPED_MYSQL_SCHEMA", "shipped") // database name
	if e != nil {
		return creds, e
	}
	x.db_user, e = getenv("SHIPPED_MYSQL_USER", "root")
	if e != nil {
		return creds, e
	}
	x.db_password, e = getenv("SHIPPED_MYSQL_PASSWORD", "shipped")
	if e != nil {
		return creds, e
	}
	return x, nil
}

// Get a (long lived) database object
func getDBObject() (db *sql.DB, e error) {
	creds, e := getdbCreds()
	if e != nil {
		log.Printf("Error getting database creds: %s", e.Error())
		return
	}

	// Create the db object
	cxn := fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, creds.db_schema)
	dbx, e := sql.Open("mysql", cxn)
	if e != nil {
		log.Printf("error getting db object: %s", e.Error())
		return
	}

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Printf("Error from db.Ping: %s", e.Error())
		return
	} else {
		log.Printf("Successfully connected to database")
	}
	return dbx, nil
}

// Create the shipped database if it does not exist
// then populate the catalog table from the json defined rows
func createDatabase() (e error) {

	var create_database string = `CREATE DATABASE IF NOT EXISTS %s`

	var create_table string = "" +
		`
		CREATE TABLE IF NOT EXISTS catalog
		(
		item_id INT PRIMARY KEY,
		name    VARCHAR(255) NOT NULL,
		description VARCHAR(255) NOT NULL,
		price   FLOAT NOT NULL,
		image   VARCHAR(255)
		)`

	var insert_table string = `INSERT IGNORE INTO catalog (item_id, name, description, price, image) VALUES (?,?,?,?,?)`

	creds, e := getdbCreds()
	if e != nil {
		return e
	}

	// Initially the shipped catalog db may not exist,
	// so connect with a database that will always exist.
	cxn := fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, "information_schema")
	dbx, e := sql.Open("mysql", cxn)
	if e != nil {
		log.Printf("error getting db object: %s", e.Error())
		return e
	}
	defer dbx.Close()

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Printf("Error from db.Ping: %s", e.Error())
		return e
	}

	// Create the database
	_, e = dbx.Exec(fmt.Sprintf(create_database, creds.db_schema))
	if e != nil {
		log.Printf("Error creating database: %s", e.Error())
		return e
	}
	dbx.Close()

	// Get new db object for shipped database.
	cxn = fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, creds.db_schema)
	dbx, e = sql.Open("mysql", cxn)
	if e != nil {
		log.Printf("error getting db object: %s", e.Error())
		return e
	}
	defer dbx.Close()

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Printf("Error from db.Ping: %s", e.Error())
		return e
	} else {
		log.Printf("Success connecting to database:  %s:********@%s/%s", creds.db_user, creds.db_host, creds.db_schema)
	}

	// Create the catalog table
	_, e = dbx.Exec(create_table)
	if e != nil {
		log.Printf("Error creating database: %s", e.Error())
		return e
	}

	// Get database rows defined as json
	var cis CatalogItems
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		log.Printf("Error reading catalog json file: %s", e.Error())
		return e
	}
	json.Unmarshal(file, &cis)

	// Get a database transaction (ensures all operations use same connection)
	tx, e := dbx.Begin()
	if e != nil {
		log.Printf("Error getting database transaction: %s", e.Error())
		return e
	}
	defer tx.Rollback()

	// Prepare insert command
	stmt, e := tx.Prepare(insert_table)
	if e != nil {
		log.Printf("Error creating prepared statement: %s", e.Error())
		return e
	}
	defer stmt.Close()

	// Populate rows of catalog table
	for _, item := range cis.Items {
		_, e = stmt.Exec(item.ItemID, item.Name, item.Description, item.Price, item.Image)
		if e != nil {
			log.Printf("Error inserting row into catalog table: %s", e.Error())
			return e
		}
	}

	e = tx.Commit()
	if e != nil {
		log.Printf("Error during transaction commit: %s", e.Error())
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
		log.Printf("Error reading database row for item %d: %s", item, e.Error())
		return ci, e
	}
	return ci, nil
}

// Get the whole catalog from the database
func getCatalog() (cat CatalogItems, e error) {

	rows, e := db.Query("SELECT  item_id, name, description, price, image FROM catalog")
	if e != nil {
		log.Printf("Error from db.Query: %s", e.Error())
		return
	}
	defer rows.Close()

	var ci CatalogItem
	var cis CatalogItems

	for rows.Next() {
		err := rows.Scan(&ci.ItemID, &ci.Name, &ci.Description, &ci.Price, &ci.Image)
		if err != nil {
			log.Printf("Error from rows.Scan: %s", err.Error())
			return
		}
		cis.Items = append(cis.Items, ci)
	}
	e = rows.Err()
	if e != nil {
		log.Printf("Error from rows: %s", e.Error())
		return
	}
	rows.Close()

	return cis, nil
}

// Catalog this will return an item or the whole list
func Catalog(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	// Load JSON File
	var catalog CatalogType
	file := loadCatalog()
	json.Unmarshal(file, &catalog)

	switch req.Method {
	//curl -X GET -H "Content-Type: application/json" http://localhost:8000/v1/catalog/3?mock=true
	case "GET":
		// Check Parameter
		uriSegments := strings.Split(req.URL.Path, "/")

		//  Get item number
		var itemNumber = 0
		if len(uriSegments) >= 3 {
			itemNumber, _ = strconv.Atoi(uriSegments[3])
		}
		// Check if mock is set true
		mock := mockCheck(req)
		if mock == true {
			if itemNumber > 0 {
				// Send catalog by item_id
				if len(catalog.Items) >= itemNumber {
					response, err := json.MarshalIndent(catalog.Items[itemNumber-1], "", "    ")
					if err != nil {
						log.Println(err)
						return
					}
					rw.WriteHeader(http.StatusAccepted)
					rw.Write([]byte(response))
					log.Println("Succesfully sent item_number:", itemNumber)
				} else {
					// item_id not found
					rw.WriteHeader(http.StatusNotFound)
					err := response(errors, http.StatusMethodNotAllowed, "Item out of index")
					rw.Write(err)
				}
			} else {
				// Send full catalog
				rw.Write([]byte(file))
			}
		} else {
			// Perform DB
			if itemNumber > 0 {
				// Send Item
				ci, e := getCatalogItem(itemNumber)
				if e != nil {
					response := fmt.Sprintf("Error from database retrieving item_id %d: %s", itemNumber, e.Error())
					rw.Write([]byte(response))
					return
				}
				response, err := json.MarshalIndent(ci, "", "    ")
				if err != nil {
					log.Printf("Error marshalling returned catalog item %s", err.Error())
					return
				}
				rw.Write([]byte(response))
				log.Printf("Succesfully sent item_number: %d", itemNumber)
			} else {
				// Send Catalog
				cis, e := getCatalog()
				if e != nil {
					log.Printf("Error getting catalog items: %s", e.Error())
					return
				}
				response, err := json.MarshalIndent(cis.Items, "", "    ")
				if err != nil {
					log.Printf("Error marshalling returned catalog item %s", err.Error())
					return
				}
				rw.Write([]byte(response))
				log.Printf("Succesfully sent %d catalog items", len(cis.Items))
			}
		}
	case "POST":
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := response(errors, http.StatusMethodNotAllowed, req.Method+" not allowed")
		rw.Write(err)
	case "PUT":
		// Update an existing record.
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := response(errors, http.StatusMethodNotAllowed, req.Method+" not allowed")
		rw.Write(err)
	case "DELETE":
		// Remove the record.
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := response(errors, http.StatusMethodNotAllowed, req.Method+" not allowed")
		rw.Write(err)
	default:
		// Give an error message.
		rw.WriteHeader(http.StatusBadRequest)
		err := response(errors, http.StatusBadRequest, "Bad request")
		rw.Write(err)
	}
}

func response(status string, code int, message string) []byte {
	resp := Response{status, code, message}
	log.Println(resp.Message)
	response, _ := json.MarshalIndent(resp, "", "    ")

	return response
}

func mockCheck(req *http.Request) bool {
	mock := req.URL.Query().Get("mock")
	if len(mock) != 0 {
		if mock == "true" {
			return true
		}
	}
	return false
}

func loadCatalog() []byte {
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		log.Printf("File error: %v\n", e)
	}
	return file
}

// HandleIndex this is the index endpoint will return 200
func HandleIndex(rw http.ResponseWriter, req *http.Request) {
	lp := path.Join("templates", "layout.html")
	fp := path.Join("templates", "index.html")

	// Note that the layout file must be the first parameter in ParseFiles
	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(rw, nil); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	// Give a success message.
	rw.WriteHeader(http.StatusOK)
	success := response(success, http.StatusOK, "Ready for request.")
	rw.Write(success)
}
