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

var DB *sql.DB // Database object

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

	db, e := createDatabase()
	if e != nil {
		log.Printf("Error creating database: %s", e.Error())
	}
	DB = db

	http.HandleFunc("/v1/catalog/", Catalog)
	http.HandleFunc("/", HandleIndex)

	// The default listening port should be set to something suitable.
	// 8889 was chosen so we could test Catalog by copying into the golang buildpack.
	listenPort := getenv("SHIPPED_CATALOG_LISTEN_PORT", "8888")
	setenv("CATALOG_PORT", listenPort)

	fmt.Println(os.Getenv("CATALOG_PORT"))

	log.Println("Listening on Port: " + listenPort)
	http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil)
}

// Get environment variable.  Return default if not set.
func getenv(name string, dflt string) (val string) {
	val = os.Getenv(name)
	if val == "" {
		val = dflt
	}
	return val
}

// Set env
func setenv(name string, val string) {
	os.Setenv(name, val)
}

func getdbCreds() (creds *dbCreds) {
	var x dbCreds

	x.db_host = getenv("SHIPPED_MYSQL_HOST", "tcp(mysql:3306)")
	x.db_schema = getenv("SHIPPED_MYSQL_SCHEMA", "shipped") // database name
	x.db_user = getenv("SHIPPED_MYSQL_USER", "root")
	x.db_password = getenv("SHIPPED_MYSQL_PASSWORD", "shipped")
	return &x
}

// Create the shipped database if it does not exist
// then populate the catalog table from the json defined rows
func createDatabase() (db *sql.DB, e error) {

	const create_database string = `CREATE DATABASE IF NOT EXISTS %s`
	const create_table string = "" +
		`
		CREATE TABLE IF NOT EXISTS catalog
		(
		item_id INT PRIMARY KEY,
		name    VARCHAR(255) NOT NULL,
		description VARCHAR(255) NOT NULL,
		price   FLOAT NOT NULL,
		image   VARCHAR(255)
		)`
	const insert_table string = `INSERT IGNORE INTO catalog (item_id, name, description, price, image) VALUES (?,?,?,?,?)`

	// Initially the shipped catalog db may not exist,
	// so connect with a database that will always exist.
	creds := getdbCreds()
	cxn := fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, "information_schema")
	dbx, e := sql.Open("mysql", cxn)
	if e != nil {
		log.Printf("Error getting db object: %s", e.Error())
		return db, e
	}

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Printf("Error from DB.Ping 1: %s", e.Error())
		return db, e
	}

	// Create the database
	_, e = dbx.Exec(fmt.Sprintf(create_database, creds.db_schema))
	if e != nil {
		log.Printf("Error creating database: %s", e.Error())
		return db, e
	}
	dbx.Close()

	// Get new db object for shipped database.
	cxn = fmt.Sprintf("%s:%s@%s/%s", creds.db_user, creds.db_password, creds.db_host, creds.db_schema)
	dbx, e = sql.Open("mysql", cxn)
	if e != nil {
		log.Printf("error getting db object: %s", e.Error())
		return db, e
	}

	// The db object does not actually connect to the database.
	// Therefore, ping the database to ensure we can connect.
	e = dbx.Ping()
	if e != nil {
		log.Printf("Error from DB.Ping 2: %s", e.Error())
		return db, e
	} else {
		log.Printf("Success connecting to database:  %s:********@%s/%s", creds.db_user, creds.db_host, creds.db_schema)
	}

	// Create the catalog table
	_, e = dbx.Exec(create_table)
	if e != nil {
		log.Printf("Error creating database: %s", e.Error())
		return db, e
	}

	// Get database rows defined as json
	var cis CatalogItems
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		log.Printf("Error reading catalog json file: %s", e.Error())
		return db, e
	}
	json.Unmarshal(file, &cis)

	// Get a database transaction (ensures all operations use same connection)
	tx, e := dbx.Begin()
	if e != nil {
		log.Printf("Error getting database transaction: %s", e.Error())
		return db, e
	}
	defer tx.Rollback()

	// Prepare insert command
	stmt, e := tx.Prepare(insert_table)
	if e != nil {
		log.Printf("Error creating prepared statement: %s", e.Error())
		return db, e
	}
	defer stmt.Close()

	// Populate rows of catalog table
	for _, item := range cis.Items {
		_, e = stmt.Exec(item.ItemID, item.Name, item.Description, item.Price, item.Image)
		if e != nil {
			log.Printf("Error inserting row into catalog table: %s", e.Error())
			return db, e
		}
	}

	e = tx.Commit()
	if e != nil {
		log.Printf("Error during transaction commit: %s", e.Error())
		return db, e
	}

	stmt.Close()
	return dbx, nil
}

// Get a single catalog row
func getCatalogItem(item int) (ci CatalogItem, e error) {
	e = DB.QueryRow("SELECT item_id, name, description, price, image FROM catalog WHERE item_id = ?", item).Scan(
		&ci.ItemID, &ci.Name, &ci.Description, &ci.Price, &ci.Image)
	if e != nil {
		log.Printf("Error reading database row for item %d: %s", item, e.Error())
		return ci, e
	}
	return ci, nil
}

// Get the whole catalog from the database
func getCatalog() (cat CatalogItems, e error) {

	rows, e := DB.Query("SELECT  item_id, name, description, price, image FROM catalog")
	if e != nil {
		log.Printf("Error from DB.Query: %s", e.Error())
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

// Delete row from catalog table
func deleteCatalogItem(item_id int) (rows int64, e error) {
	const delete_sql = `DELETE FROM catalog WHERE item_id = ?`

	res, err := DB.Exec(delete_sql, item_id)
	if err != nil {
		log.Printf("Error deleting row %d:  %s", item_id, err.Error())
		return 0, err
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error fetching count of rows deleted: %s", err.Error())
		return 0, err
	}
	return rowCnt, nil
}

// Add an item to the catalog
func addCatalogItem(req *http.Request) (e error) {
	const insert_sql = `INSERT INTO catalog (item_id, name, description, price, image) VALUES (?,?,?,?,?)`
	req.ParseForm()

	item_id := strings.Join(req.Form["item_id"], "")
	name := strings.Join(req.Form["name"], "")
	desc := strings.Join(req.Form["description"], "")
	price := strings.Join(req.Form["price"], "")
	image := strings.Join(req.Form["image"], "")

	_, err := DB.Exec(insert_sql, item_id, name, desc, price, image)
	if err != nil {
		log.Printf("Error inserting row: %s", err.Error())
		return err
	}
	return nil
}

func updateCatalogItem(req *http.Request) (e error) {
	const update_sql = `UPDATE catalog SET name=?, description=?, price=?, image=? WHERE item_id=?`
	var ci CatalogItem

	// Get existing item so we can update the fields that changed
	itemNumber := getItemNumber(req)
	ci, e = getCatalogItem(itemNumber)
	if e != nil {
		log.Printf("Error getting row for update: %s", e.Error())
		return e
	}

	name := ci.Name
	desc := ci.Description
	price := ci.Price
	image := ci.Image

	req.ParseForm()

	if len(req.Form["name"]) > 0 {
		name = strings.Join(req.Form["name"], "")
	}
	if len(req.Form["description"]) > 0 {
		desc = strings.Join(req.Form["description"], "")
	}
	if len(req.Form["image"]) > 0 {
		desc = strings.Join(req.Form["image"], "")
	}
	if len(req.Form["price"]) > 0 {
		desc = strings.Join(req.Form["price"], "")
	}
	_, e = DB.Exec(update_sql, name, desc, price, image, itemNumber)
	if e != nil {
		log.Printf("Error updating row: %s", e.Error())
		return e
	}
	return nil
}

//  Get item number
func getItemNumber(req *http.Request) int {
	var itemNumber = 0
	uriSegments := strings.Split(req.URL.Path, "/")
	if len(uriSegments) >= 3 {
		itemNumber, _ = strconv.Atoi(uriSegments[3])
	}
	return itemNumber
}

// Catalog this will return an item or the whole list
func Catalog(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Load JSON File
	var catalog CatalogType
	file := loadCatalog()
	json.Unmarshal(file, &catalog)

	switch req.Method {
	//curl -X GET -H "Content-Type: application/json" http://localhost:8888/v1/catalog/3?mock=true
	case "GET":

		itemNumber := getItemNumber(req)

		// Check if mock is set true
		mock := mockCheck(req)
		if mock == true {
			if itemNumber > 0 {
				// Send catalog by item_id
				if len(catalog.Items) >= itemNumber {
					resp, err := json.MarshalIndent(catalog.Items[itemNumber-1], "", "    ")
					if err != nil {
						log.Println(err)
						return
					}
					rw.WriteHeader(http.StatusAccepted)
					rw.Write([]byte(resp))
					log.Println("Succesfully sent item_number:", itemNumber)
				} else {
					// item_id not found
					rw.WriteHeader(http.StatusNotFound)
					err := response(errors, http.StatusMethodNotAllowed, "Item out of index")
					rw.Write(err)
				}
			} else {
				// Send full catalog
				log.Println("Succesfully sent full catalog.")
				rw.Write([]byte(file))
			}
		} else { // No mock. Use MySQL DB.
			if itemNumber > 0 { // Send single Item
				ci, e := getCatalogItem(itemNumber)
				if e != nil {
					rw.WriteHeader(http.StatusBadRequest)
					rw.Write(response(errors, http.StatusBadRequest,
						fmt.Sprintf("Error from database retrieving item_id %d: %s", itemNumber, e.Error())))
					return
				}
				resp, err := json.MarshalIndent(ci, "", "    ")
				if err != nil {
					log.Printf("Error marshalling returned catalog item %s", err.Error())
					return
				}
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(resp))
				log.Printf("Succesfully sent item_number: %d", itemNumber)
			} else { // Send whole Catalog
				cis, e := getCatalog()
				if e != nil {
					log.Printf("Error getting catalog items: %s", e.Error())
					return
				}
				resp, err := json.MarshalIndent(cis.Items, "", "    ")
				if err != nil {
					log.Printf("Error marshalling returned catalog item %s", err.Error())
					return
				}
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(resp))
				log.Printf("Succesfully sent %d catalog items", len(cis.Items))
			}
		}

	case "POST": // Create new record
		itemNumber := getItemNumber(req)
		if itemNumber > 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write(response(errors, http.StatusBadRequest, fmt.Sprintf("Item number must not appear on URL")))
			return
		}
		err := addCatalogItem(req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write(response(errors, http.StatusBadRequest, fmt.Sprintf("Error adding item to catalog: %s", err.Error())))
			return
		}
		rw.WriteHeader(http.StatusCreated)
		rw.Write(response(success, http.StatusCreated, ""))
		return

	case "PUT": // Update existing record
		itemNumber := getItemNumber(req)
		if itemNumber <= 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write(response(errors, http.StatusBadRequest, fmt.Sprintf("Item number must appear on URL")))
			return
		}
		err := updateCatalogItem(req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write(response(errors, http.StatusBadRequest, fmt.Sprintf("Error updating item in catalog: %s", err.Error())))
			return
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write(response(success, http.StatusOK, ""))
		return

	case "DELETE": // Remove record.
		itemNumber := getItemNumber(req)
		if itemNumber <= 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write(response(errors, http.StatusBadRequest, fmt.Sprintf("Invalid item number: %d", itemNumber)))
			return
		}
		rowCnt, err := deleteCatalogItem(itemNumber)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write(response(errors, http.StatusBadRequest, fmt.Sprintf("Error deleting item_id %d:  %s", itemNumber, err.Error())))
			return
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write(response(success, http.StatusOK, fmt.Sprintf("Rows affected: %d", rowCnt)))

	default:
		// Give an error message.
		rw.WriteHeader(http.StatusMethodNotAllowed)
		rw.Write(response(errors, http.StatusMethodNotAllowed, req.Method+" not allowed"))
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
