package main

import (
	"html/template"
	"log"
	"net/http"
)

type Room struct {
	ID          int
	Name        string
	Description string
	Price       int
	Capacity    int
}

var rooms = []Room{
	{
		ID:          1,
		Name:        "Deluxe King Room",
		Description: "A spacious room with a king-size bed and modern amenities.",
		Price:       85000,
		Capacity:    2,
	},
	{
		ID:          2,
		Name:        "Executive Suite",
		Description: "A premium suite designed for guests who want extra comfort.",
		Price:       140000,
		Capacity:    3,
	},
	{
		ID:          3,
		Name:        "Presidential Suite",
		Description: "Our most luxurious accommodation with an exceptional guest experience.",
		Price:       250000,
		Capacity:    4,
	},
}

func main() {
	tmpl := template.Must(
		template.ParseFiles("templates/index.html"),
	)

	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := struct {
			Rooms []Room
		}{
			Rooms: rooms,
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Println("Template error:", err)
			http.Error(
				w,
				"Unable to render page",
				http.StatusInternalServerError,
			)
		}
	})

	log.Println("Hotel platform running at http://localhost:8081")

	log.Fatal(
		http.ListenAndServe(":8081", nil),
	)
}
