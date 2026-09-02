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
	http.HandleFunc("/book", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Unable to process booking", http.StatusBadRequest)
			return
		}

		checkIn := r.FormValue("check_in")
		checkOut := r.FormValue("check_out")
		guests := r.FormValue("guests")
		roomID := r.FormValue("room")

		log.Println("New booking received:")
		log.Println("Check-in:", checkIn)
		log.Println("Check-out:", checkOut)
		log.Println("Guests:", guests)
		log.Println("Room:", roomID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Booking received successfully"))
	})

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
