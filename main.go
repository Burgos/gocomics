package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type ComicBook struct {
	Status bool
	Number int
	Title  string
	Hero   string
	Image  []byte
}

var db *sql.DB

func comicAtId(id int, status bool) ComicBook {
	return ComicBook{Status: status, Number: id, Title: "Lov na coveka"}
}

func getBooks() []ComicBook {
	books := make([]ComicBook, 0)
	rows, err := db.Query("SELECT id, junak, naslov, stanje FROM zlatna_serija")
	if err != nil {
		panic(err)
	}

	var book ComicBook
	for rows.Next() {
		err = rows.Scan(&book.Number, &book.Hero, &book.Title, &book.Status)
		if err != nil {
			panic(err)
		}
		books = append(books, book)
	}

	return books
}

func handler(w http.ResponseWriter, r *http.Request) {
	books := getBooks()

	t, err := template.ParseFiles("table.html")
	if err != nil {
		panic(err)
	}
	err = t.Execute(w, books)
	if err != nil {
		panic(err)
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	img_id := r.URL.Path[len("/image/") : len(r.URL.Path)-len(".jpg")]
	row := db.QueryRow("SELECT slicica FROM zlatna_serija WHERE id = $1", img_id)

	var image []byte
	err := row.Scan(&image)
	if err != nil {
		panic(err)
	}
	key := "zlatna_serija_small" + img_id
	e := `"` + key + `"`
	w.Header().Set("Etag", e)
	w.Header().Set("Cache-Control", "max-age=2592000")

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, e) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Write(image)
}

func fullImageHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("full path: %s", r.URL.Path)
	img_id := r.URL.Path[len("/full_image/") : len(r.URL.Path)-len(".jpg")]
	row := db.QueryRow("SELECT slika FROM zlatna_serija WHERE id = $1", img_id)

	var image []byte
	err := row.Scan(&image)
	if err != nil {
		panic(err)
	}
	key := "zlatna_serija_full" + img_id
	e := `"` + key + `"`
	w.Header().Set("Etag", e)
	w.Header().Set("Cache-Control", "max-age=2592000")

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, e) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Write(image)
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	stmt, err := db.Prepare("UPDATE zlatna_serija SET stanje = CASE stanje WHEN 1 THEN 0 ELSE 1 END WHERE id = ?")
	if err != nil {
		panic(err)
	}
	id := r.URL.Path[len("/toggle_status/"):]
	_, err = stmt.Exec(id)
	if err != nil {
		panic(err)
	}
	http.Redirect(w, r, "/", 302)
}

func main() {
	type Config struct {
		Database    string
		BindAddress string
	}

	// https://stackoverflow.com/questions/16465705/how-to-handle-configuration-in-go
	file, err := os.Open("conf.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	db, err = sql.Open("sqlite3", config.Database)
	if err != nil {
		panic(err)
	}
	log.Printf("Loaded database from %s", config.Database)

	http.HandleFunc("/", handler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/toggle_status/", toggleHandler)
	http.HandleFunc("/full_image/", fullImageHandler)

	log.Printf("Listening on %s", config.BindAddress)
	log.Fatal(http.ListenAndServe(config.BindAddress, nil))
}
