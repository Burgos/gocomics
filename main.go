package main

import (
	"log"
	"net/http"
	"html/template"
	"database/sql"
    _ "github.com/mattn/go-sqlite3"
)

type ComicBook struct {
	Status bool
	Number int
	Title string
	Hero string
	Image []byte
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
		if err != nil { panic(err) }
		books = append(books, book)
	}
	
	return books
}

func handler(w http.ResponseWriter, r *http.Request) {
	books := getBooks()
	
	
	t, err := template.ParseFiles("table.html")
	if err != nil { panic(err) }
	err = t.Execute(w, books)
	if err != nil { panic(err) }
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	row := db.QueryRow("SELECT slika FROM zlatna_serija WHERE id = $1", r.URL.Path[len("/image/"):])
	
	var image []byte
	err := row.Scan(&image)
	if err != nil { panic(err) }
	w.Write(image)
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./stripovi.s3db")
	if err != nil { panic(err) }
	http.HandleFunc("/", handler)
	http.HandleFunc("/image/", imageHandler)
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}