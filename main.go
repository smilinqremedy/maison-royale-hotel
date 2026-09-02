package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"maison-royale/database"
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
	db := database.Connect()
	defer db.Close()

	tmpl := template.Must(
		template.ParseFiles(
			"templates/index.html",
			"templates/confirmation.html",
		),
	)

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
		guestsText := r.FormValue("guests")
		roomIDText := r.FormValue("room")

		if checkIn == "" || checkOut == "" {
			http.Error(w, "Check-in and check-out dates are required", http.StatusBadRequest)
			return
		}

		guests, err := strconv.Atoi(guestsText)
		if err != nil || guests < 1 {
			http.Error(w, "Invalid number of guests", http.StatusBadRequest)
			return
		}

		roomID, err := strconv.Atoi(roomIDText)
		if err != nil {
			http.Error(w, "Invalid room", http.StatusBadRequest)
			return
		}

		var selectedRoom Room
		foundRoom := false

		for _, room := range rooms {
			if room.ID == roomID {
				selectedRoom = room
				foundRoom = true
				break
			}
		}

		if !foundRoom {
			http.Error(w, "Room not found", http.StatusBadRequest)
			return
		}

		if guests > selectedRoom.Capacity {
			http.Error(w, "Too many guests for this room", http.StatusBadRequest)
			return
		}

		query := `
			INSERT INTO bookings (check_in, check_out, guests, room_id)
			VALUES (?, ?, ?, ?)
		`

		result, err := db.Exec(
			query,
			checkIn,
			checkOut,
			guests,
			roomID,
		)
		if err != nil {
			log.Println("Failed to save booking:", err)
			http.Error(
				w,
				"Unable to save booking",
				http.StatusInternalServerError,
			)
			return
		}

		bookingID, err := result.LastInsertId()
		if err != nil {
			log.Println("Failed to get booking ID:", err)
			http.Error(
				w,
				"Unable to get booking ID",
				http.StatusInternalServerError,
			)
			return
		}

		log.Println("Booking saved successfully")
		log.Println("Booking ID:", bookingID)
		log.Println("Check-in:", checkIn)
		log.Println("Check-out:", checkOut)
		log.Println("Guests:", guests)
		log.Println("Room:", selectedRoom.Name)

		confirmationData := struct {
			ID       int64
			RoomName string
			CheckIn  string
			CheckOut string
			Guests   int
			Price    int
		}{
			ID:       bookingID,
			RoomName: selectedRoom.Name,
			CheckIn:  checkIn,
			CheckOut: checkOut,
			Guests:   guests,
			Price:    selectedRoom.Price,
		}

		if err := tmpl.ExecuteTemplate(
			w,
			"confirmation.html",
			confirmationData,
		); err != nil {
			log.Println("Confirmation template error:", err)
			http.Error(
				w,
				"Unable to show confirmation",
				http.StatusInternalServerError,
			)
			return
		}
	})

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

		if err := tmpl.ExecuteTemplate(
			w,
			"index.html",
			data,
		); err != nil {
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
