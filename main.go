package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

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
			"templates/unavailable.html",
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

		// Make sure both dates were provided.
		if checkIn == "" || checkOut == "" {
			http.Error(
				w,
				"Check-in and check-out dates are required",
				http.StatusBadRequest,
			)
			return
		}

		// Validate check-in date.
		checkInDate, err := time.Parse("2006-01-02", checkIn)
		if err != nil {
			http.Error(w, "Invalid check-in date", http.StatusBadRequest)
			return
		}

		// Validate check-out date.
		checkOutDate, err := time.Parse("2006-01-02", checkOut)
		if err != nil {
			http.Error(w, "Invalid check-out date", http.StatusBadRequest)
			return
		}

		// Check-out must be after check-in.
		if !checkOutDate.After(checkInDate) {
			http.Error(
				w,
				"Check-out date must be after check-in date",
				http.StatusBadRequest,
			)
			return
		}

		// Convert guests to an integer.
		guests, err := strconv.Atoi(guestsText)
		if err != nil || guests < 1 {
			http.Error(
				w,
				"Invalid number of guests",
				http.StatusBadRequest,
			)
			return
		}

		// Convert room ID to an integer.
		roomID, err := strconv.Atoi(roomIDText)
		if err != nil {
			http.Error(
				w,
				"Invalid room",
				http.StatusBadRequest,
			)
			return
		}

		// Find the selected room.
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
			http.Error(
				w,
				"Room not found",
				http.StatusBadRequest,
			)
			return
		}

		// Make sure the number of guests fits the room.
		if guests > selectedRoom.Capacity {
			http.Error(
				w,
				"Too many guests for this room",
				http.StatusBadRequest,
			)
			return
		}

		// Check whether the room is already booked
		// during any part of the requested stay.
		var existingBooking int

		availabilityQuery := `
			SELECT COUNT(*)
			FROM bookings
			WHERE room_id = ?
			AND check_in < ?
			AND check_out > ?
		`

		err = db.QueryRow(
			availabilityQuery,
			roomID,
			checkOut,
			checkIn,
		).Scan(&existingBooking)

		if err != nil {
			log.Println("Failed to check room availability:", err)

			http.Error(
				w,
				"Unable to check room availability",
				http.StatusInternalServerError,
			)
			return
		}

		// If a booking overlaps, show the professional
		// room unavailable page.
		if existingBooking > 0 {
			unavailableData := struct {
				RoomName string
				CheckIn  string
				CheckOut string
			}{
				RoomName: selectedRoom.Name,
				CheckIn:  checkIn,
				CheckOut: checkOut,
			}

			if err := tmpl.ExecuteTemplate(
				w,
				"unavailable.html",
				unavailableData,
			); err != nil {
				log.Println("Unavailable template error:", err)

				http.Error(
					w,
					"Unable to show availability message",
					http.StatusInternalServerError,
				)
			}

			return
		}

		// The room is available, so create the booking.
		query := `
			INSERT INTO bookings (
				check_in,
				check_out,
				guests,
				room_id
			)
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

		// Data sent to the confirmation page.
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

	// Serve CSS and JavaScript files.
	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	// Homepage.
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
