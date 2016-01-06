package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
)

// https://toyshop.com/api/catalog/list - List all items
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome, %!", r.URL.Path[1:])
}

func main() {
	http.HandleFunc("/api/catalog/list/", list)
	http.HandleFunc("/", foo)

	http.ListenAndServe(":8000", nil)
}

type Login struct {
	Login string
}

func foo(w http.ResponseWriter, r *http.Request) {
	// Assuming you want to serve a photo at 'images/foo.png'
	fp := path.Join("images/lightsaber.jpg")
	http.ServeFile(w, r, fp)
}

func list(rw http.ResponseWriter, req *http.Request) {
	// body, err := ioutil.ReadAll(req.Body)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println(string(body))
	//
	// var t Login
	// err = json.Unmarshal(body, &t)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Println(t.Login)
	fmt.Fprintf(rw, "Welcome, %!", req.URL.Path[1:])

	// All Items
	// TODO
	file, e := ioutil.ReadFile("./catalog.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		os.Exit(1)
	}
	rw.Write([]byte(file))

	log.Println("Sent Catalog")

	//
	//else
}
