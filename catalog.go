package main

import (
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

func main() {
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

var jsontype CatalogType

// Catalog this will return an item or the whole list
func Catalog(w http.ResponseWriter, r *http.Request) {
	// TODO: ADD MOCK Feature and DB to fetch for backend

	// Load JSON File
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}

	json.Unmarshal(file, &jsontype)

	switch r.Method {
	case "GET":
		// Check Parameter
		uriSegments := strings.Split(r.URL.Path, "/")
		var itemNumber = 0
		if len(uriSegments) >= 3 {
			itemNumber, _ = strconv.Atoi(uriSegments[3])
		}
		mock := r.URL.Query().Get("mock")
		if len(mock) != 0 {
			if mock == "true" {
				if itemNumber > 0 {
					// Send catalog by item_id
					if len(jsontype.Items) >= itemNumber {
						response, err := json.MarshalIndent(jsontype.Items[itemNumber-1], "", "    ")
						if err != nil {
							fmt.Println(err)
							return
						}
						w.Write([]byte(response))
						log.Println("Succesfully sent item_number:", itemNumber)
					}
				} else {
					// Send full catalog
					w.Write([]byte(file))
				}
			}
		} else {
			// Perform DB Query

		}
	case "POST":

	case "PUT":
		// Update an existing record.
	case "DELETE":
		// Remove the record.
	default:
		// Give an error message.
	}
}
