package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type ComicBook struct {
	Id      int
	Status  bool
	Number  int
	Title   string
	Hero    string
	Edicija string
	Image   []byte
}

var db *sql.DB

func comicAtId(id int, status bool) ComicBook {
	return ComicBook{Status: status, Number: id, Title: "Lov na coveka"}
}

func getBooks(edicija string) []ComicBook {
	books := make([]ComicBook, 0)
	log.Printf("edicija = '%s'", edicija)
	rows, err := db.Query("SELECT id, broj, junak, naslov, stanje FROM comics WHERE edicija = $1 ORDER by broj", edicija)
	if err != nil {
		panic(err)
	}

	var book ComicBook
	for rows.Next() {
		err = rows.Scan(&book.Id, &book.Number, &book.Hero, &book.Title, &book.Status)
		if err != nil {
			panic(err)
		}
		book.Edicija = edicija
		books = append(books, book)
	}

	return books
}

func checkAccess(user string, pass string) bool {
	if user == os.Getenv("USER") && CheckPasswordHash(pass, os.Getenv("password_hash")) {
		return true
	}
	return false
}

func handler(edicija string, w http.ResponseWriter, r *http.Request) {
	user, pass, _ := r.BasicAuth()
	if !checkAccess(user, pass) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Stripovi\"")
		http.Error(w, "Unauthorized.", 401)
		return
	}

	books := getBooks(edicija)

	t, err := template.ParseFiles("table.html")
	if err != nil {
		panic(err)
	}
	err = t.Execute(w, books)
	if err != nil {
		panic(err)
	}
}

func imageHandler(edicija string, w http.ResponseWriter, r *http.Request) {
	img_id := r.URL.Path[len("/"+edicija+"/image/") : len(r.URL.Path)-len(".jpg")]
	row := db.QueryRow("SELECT slicica FROM comics WHERE edicija = $1 AND id = $2", edicija, img_id)

	var image []byte
	err := row.Scan(&image)
	if err != nil {
		panic(err)
	}
	key := edicija + "_small" + img_id
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

func fullImageHandler(edicija string, w http.ResponseWriter, r *http.Request) {
	log.Printf("full path: %s", r.URL.Path)
	img_id := r.URL.Path[len("/"+edicija+"/full_image/") : len(r.URL.Path)-len(".jpg")]
	row := db.QueryRow("SELECT slika FROM comics WHERE edicija = $1 AND id = $2", edicija, img_id)

	var image []byte
	err := row.Scan(&image)
	if err != nil {
		panic(err)
	}
	key := edicija + "_full" + img_id
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

func toggleHandler(edicija string, w http.ResponseWriter, r *http.Request) {
	user, pass, _ := r.BasicAuth()
	if !checkAccess(user, pass) {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Stripovi\"")
		http.Error(w, "Unauthorized.", 401)
		return
	}

	query := "UPDATE comics SET stanje = CASE stanje WHEN TRUE THEN FALSE ELSE TRUE END WHERE edicija = $1 AND id = $2"
	stmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	id := r.URL.Path[len("/"+edicija+"/toggle_status/"):]
	_, err = stmt.Exec(edicija, id)
	if err != nil {
		log.Printf("Failed executing %s for %s\n", query, id)
		panic(err)
	}
	http.Redirect(w, r, "/"+edicija, 302)
}

func main() {
	type Config struct {
		Address  string
		Port     string
		Postgres string
	}

	var config Config
	config.Address = os.Getenv("ADDRESS")
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

	edicije := []string{"zlatna_serija", "zagor_redovan", "teks_redovan", "zagor_knjiga", "teks_knjiga"}

	for _, ed := range edicije {
		edicija := ed
		http.HandleFunc("/"+edicija,
			func(w http.ResponseWriter, r *http.Request) {
				handler(edicija, w, r)
			})

		http.HandleFunc("/"+edicija+"/image/",
			func(w http.ResponseWriter, r *http.Request) {
				imageHandler(edicija, w, r)
			})
		http.HandleFunc("/"+edicija+"/toggle_status/",
			func(w http.ResponseWriter, r *http.Request) {
				toggleHandler(edicija, w, r)
			})
		http.HandleFunc("/"+edicija+"/full_image/",
			func(w http.ResponseWriter, r *http.Request) {
				fullImageHandler(edicija, w, r)
			})
	}

	log.Printf("Listening on %s", config.Port)
	log.Fatal(http.ListenAndServe(config.Address+":"+config.Port, nil))
}
