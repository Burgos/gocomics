package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type ComicBook struct {
	Id     int
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
	rows, err := db.Query("SELECT id, broj, junak, naslov, stanje FROM comics ORDER by broj")
	if err != nil {
		panic(err)
	}

	var book ComicBook
	for rows.Next() {
		err = rows.Scan(&book.Id, &book.Number, &book.Hero, &book.Title, &book.Status)
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
	row := db.QueryRow("SELECT slicica FROM comics WHERE id = $1", img_id)

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
	row := db.QueryRow("SELECT slika FROM comics WHERE id = $1", img_id)

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
	query := "UPDATE comics SET stanje = CASE stanje WHEN TRUE THEN FALSE ELSE TRUE END WHERE id = $1"
	stmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	id := r.URL.Path[len("/toggle_status/"):]
	_, err = stmt.Exec(id)
	if err != nil {
		log.Printf("Failed executing %s for %s\n", query, id)
		panic(err)
	}
	http.Redirect(w, r, "/", 302)
}

func main() {
	type Config struct {
		Port     string
		Postgres string
	}

	var config Config
	config.Port = os.Getenv("PORT")
	config.Postgres = os.Getenv("DATABASE_URL")

	var err error
	db, err = sql.Open("postgres", config.Postgres)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	log.Printf("Successfully connected to postgres")

	http.HandleFunc("/", handler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/toggle_status/", toggleHandler)
	http.HandleFunc("/full_image/", fullImageHandler)

	log.Printf("Listening on %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}
