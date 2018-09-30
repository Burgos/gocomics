package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

type ComicBook struct {
	Status     bool
	Number     int
	Title      string
	Hero       string
	Image      []byte
	SmallImage []byte
}

var db *sql.DB

func getBooks() []ComicBook {
	books := make([]ComicBook, 0)
	rows, err := db.Query("SELECT id, junak, naslov, stanje, slika, slicica FROM zlatna_serija")
	if err != nil {
		panic(err)
	}

	var book ComicBook
	for rows.Next() {
		err = rows.Scan(&book.Number, &book.Hero, &book.Title, &book.Status,
			&book.Image, &book.SmallImage)
		if err != nil {
			panic(err)
		}
		books = append(books, book)
	}

	return books
}

func main() {
	type Config struct {
		Database         string
		BindAddress      string
		Postgres         string
		PostgresPort     string
		PostgresUser     string
		PostgresPassword string
		PostgresDatabase string
		PostgresTable    string
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

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		config.Postgres, config.PostgresPort, config.PostgresUser,
		config.PostgresPassword, config.PostgresDatabase)

	pgdb, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer pgdb.Close()

	err = pgdb.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to postgres")

	books := getBooks()
	for i := range books {
		if _, err := pgdb.Exec("INSERT INTO comics (broj, junak, naslov, stanje, slicica, slika, kolekcija) VALUES ($1, $2, $3, $4, $5, $6, 1)",
			books[i].Number, books[i].Hero, books[i].Title, books[i].Status, books[i].SmallImage, books[i].Image); err != nil {
			log.Fatalf("Error inserting comic: %s", err)
		}
	}
}
