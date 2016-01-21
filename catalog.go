package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
)

type CatalogType struct {
	Items []struct {
		Image       string  `json:"image"`
		ItemID      int     `json:"item_id"`
		Name        string  `json:"name"`
		Descrpition string  `json:"description"`
		Price       float64 `json:"price"`
	} `json:"items"`
}

type Response struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const error = "ERROR"
const success = "SUCCESS"

func main() {
	http.HandleFunc("/v1/catalog/", Catalog)
	http.HandleFunc("/", HandleIndex)

	http.ListenAndServe(":8000", nil)
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
		mock := req.URL.Query().Get("mock")
		if len(mock) != 0 {
			if mock == "true" {
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
						err := response(error, http.StatusMethodNotAllowed, "Item out of index")
						rw.Write(err)
					}
				} else {
					// Send full catalog
					rw.Write([]byte(file))
				}
			}
		} else {
			// Perform DB Query

		}
	case "POST":
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := response(error, http.StatusMethodNotAllowed, req.Method+" not allowed")
		rw.Write(err)
	case "PUT":
		// Update an existing record.
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := response(error, http.StatusMethodNotAllowed, req.Method+" not allowed")
		rw.Write(err)
	case "DELETE":
		// Remove the record.
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := response(error, http.StatusMethodNotAllowed, req.Method+" not allowed")
		rw.Write(err)
	default:
		// Give an error message.
		rw.WriteHeader(http.StatusBadRequest)
		err := response(error, http.StatusBadRequest, "Bad request")
		rw.Write(err)
	}
}

func response(status string, code int, message string) []byte {
	error := Response{status, code, message}
	log.Println(error.Message)
	response, _ := json.MarshalIndent(error, "", "    ")

	return response
}

func loadCatalog() []byte {
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		log.Printf("File error: %v\n", e)
		os.Exit(1)
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
