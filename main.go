package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	"maison-royale/database"
)

type Room struct {
	ID          int
	Name        string
	Description string
	Price       int
	Capacity    int
}

type Booking struct {
	ID       int
	RoomName string
	CheckIn  string
	CheckOut string
	Guests   int
	Price    int
	Nights   int
	Total    int
}

type RoomAvailability struct {
	Name      string
	Available bool
}

type DashboardData struct {
	TotalBookings  int
	TotalRooms     int
	AvailableRooms int
	Bookings       []Booking
	CheckIn        string
	CheckOut       string
	Availability   []RoomAvailability
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

const sessionCookieName = "maison_admin_session"

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)

	if err != nil {
		return false
	}

	return cookie.Value == "authenticated"
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			http.Redirect(
				w,
				r,
				"/admin/login",
				http.StatusSeeOther,
			)
			return
		}

		next(w, r)
	}
}

func loadDashboardData(db *sql.DB) (DashboardData, error) {
	var totalBookings int

	err := db.QueryRow(
		"SELECT COUNT(*) FROM bookings",
	).Scan(&totalBookings)

	if err != nil {
		return DashboardData{}, err
	}

	bookings := []Booking{}

	rows, err := db.Query(`
		SELECT
			bookings.id,
			bookings.check_in,
			bookings.check_out,
			bookings.guests,
			bookings.room_id
		FROM bookings
		ORDER BY bookings.id DESC
	`)

	if err != nil {
		return DashboardData{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var booking Booking
		var roomID int

		err := rows.Scan(
			&booking.ID,
			&booking.CheckIn,
			&booking.CheckOut,
			&booking.Guests,
			&roomID,
		)

		if err != nil {
			return DashboardData{}, err
		}

		for _, room := range rooms {
			if room.ID == roomID {
				booking.RoomName = room.Name
				break
			}
		}

		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return DashboardData{}, err
	}

	availableRooms := len(rooms)

	for _, room := range rooms {
		var roomBookings int

		err := db.QueryRow(`
			SELECT COUNT(*)
			FROM bookings
			WHERE room_id = ?
			AND check_out > date('now')
		`, room.ID).Scan(&roomBookings)

		if err != nil {
			return DashboardData{}, err
		}

		if roomBookings > 0 {
			availableRooms--
		}
	}

	return DashboardData{
		TotalBookings:  totalBookings,
		TotalRooms:     len(rooms),
		AvailableRooms: availableRooms,
		Bookings:       bookings,
	}, nil
}

func main() {
	db := database.Connect()
	defer db.Close()

	tmpl := template.Must(
		template.ParseFiles(
			"templates/index.html",
			"templates/confirmation.html",
			"templates/unavailable.html",
			"templates/admin/dashboard.html",
			"templates/admin/login.html",
		),
	)

	// =========================================================
	// ADMIN LOGIN
	// =========================================================

	http.HandleFunc("/admin/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if isAuthenticated(r) {
				http.Redirect(
					w,
					r,
					"/admin",
					http.StatusSeeOther,
				)
				return
			}

			data := struct {
				Error string
			}{
				Error: "",
			}

			if err := tmpl.ExecuteTemplate(
				w,
				"login.html",
				data,
			); err != nil {
				log.Println("Login template error:", err)

				http.Error(
					w,
					"Unable to render login page",
					http.StatusInternalServerError,
				)
			}

			return
		}

		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Unable to process login",
				http.StatusBadRequest,
			)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		adminUsername := os.Getenv("ADMIN_USERNAME")
		adminPasswordHash := os.Getenv("ADMIN_PASSWORD_HASH")

		if adminUsername == "" || adminPasswordHash == "" {
			log.Println("Admin credentials are not configured")

			http.Error(
				w,
				"Admin authentication is not configured",
				http.StatusInternalServerError,
			)

			return
		}

		if username != adminUsername {
			data := struct {
				Error string
			}{
				Error: "Invalid username or password.",
			}

			if err := tmpl.ExecuteTemplate(
				w,
				"login.html",
				data,
			); err != nil {
				log.Println("Login template error:", err)
			}

			return
		}

		err := bcrypt.CompareHashAndPassword(
			[]byte(adminPasswordHash),
			[]byte(password),
		)

		if err != nil {
			data := struct {
				Error string
			}{
				Error: "Invalid username or password.",
			}

			if err := tmpl.ExecuteTemplate(
				w,
				"login.html",
				data,
			); err != nil {
				log.Println("Login template error:", err)
			}

			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "authenticated",
			Path:     "/admin",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
			MaxAge:   60 * 60 * 8,
		})

		log.Println("Admin login successful")

		http.Redirect(
			w,
			r,
			"/admin",
			http.StatusSeeOther,
		)
	})

	// =========================================================
	// ADMIN LOGOUT
	// =========================================================

	http.HandleFunc("/admin/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/admin",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
			MaxAge:   -1,
		})

		log.Println("Admin logged out")

		http.Redirect(
			w,
			r,
			"/admin/login",
			http.StatusSeeOther,
		)
	})

	// =========================================================
	// ADMIN CANCEL BOOKING
	// =========================================================

	http.HandleFunc("/admin/bookings/cancel", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Unable to process cancellation",
				http.StatusBadRequest,
			)
			return
		}

		bookingID, err := strconv.Atoi(
			r.FormValue("booking_id"),
		)

		if err != nil || bookingID <= 0 {
			http.Error(
				w,
				"Invalid booking ID",
				http.StatusBadRequest,
			)
			return
		}

		result, err := db.Exec(
			"DELETE FROM bookings WHERE id = ?",
			bookingID,
		)

		if err != nil {
			log.Println(
				"Booking cancellation error:",
				err,
			)

			http.Error(
				w,
				"Unable to cancel booking",
				http.StatusInternalServerError,
			)

			return
		}

		rowsAffected, err := result.RowsAffected()

		if err != nil {
			log.Println(
				"Unable to verify booking cancellation:",
				err,
			)

			http.Error(
				w,
				"Unable to verify cancellation",
				http.StatusInternalServerError,
			)

			return
		}

		if rowsAffected == 0 {
			http.Error(
				w,
				"Booking not found",
				http.StatusNotFound,
			)

			return
		}

		log.Printf(
			"Booking #%d cancelled successfully",
			bookingID,
		)

		http.Redirect(
			w,
			r,
			"/admin",
			http.StatusSeeOther,
		)
	}))

	// =========================================================
	// ADMIN DASHBOARD
	// =========================================================

	http.HandleFunc("/admin", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		data, err := loadDashboardData(db)

		if err != nil {
			log.Println(
				"Failed to load dashboard:",
				err,
			)

			http.Error(
				w,
				"Unable to load dashboard",
				http.StatusInternalServerError,
			)

			return
		}

		if err := tmpl.ExecuteTemplate(
			w,
			"dashboard.html",
			data,
		); err != nil {
			log.Println(
				"Dashboard template error:",
				err,
			)

			http.Error(
				w,
				"Unable to render dashboard",
				http.StatusInternalServerError,
			)
		}
	}))

	// =========================================================
	// ADMIN ROOM AVAILABILITY
	// =========================================================

	http.HandleFunc("/admin/availability", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		checkIn := r.URL.Query().Get("check_in")
		checkOut := r.URL.Query().Get("check_out")

		if checkIn == "" || checkOut == "" {
			http.Error(
				w,
				"Check-in and check-out dates are required",
				http.StatusBadRequest,
			)
			return
		}

		checkInDate, err := time.Parse(
			"2006-01-02",
			checkIn,
		)

		if err != nil {
			http.Error(
				w,
				"Invalid check-in date",
				http.StatusBadRequest,
			)
			return
		}

		checkOutDate, err := time.Parse(
			"2006-01-02",
			checkOut,
		)

		if err != nil {
			http.Error(
				w,
				"Invalid check-out date",
				http.StatusBadRequest,
			)
			return
		}

		if !checkOutDate.After(checkInDate) {
			http.Error(
				w,
				"Check-out date must be after check-in date",
				http.StatusBadRequest,
			)
			return
		}

		availability := []RoomAvailability{}

		for _, room := range rooms {
			var existingBooking int

			err := db.QueryRow(`
				SELECT COUNT(*)
				FROM bookings
				WHERE room_id = ?
				AND check_in < ?
				AND check_out > ?
			`,
				room.ID,
				checkOut,
				checkIn,
			).Scan(&existingBooking)

			if err != nil {
				log.Println(
					"Failed to check room availability:",
					err,
				)

				http.Error(
					w,
					"Unable to check room availability",
					http.StatusInternalServerError,
				)

				return
			}

			availability = append(
				availability,
				RoomAvailability{
					Name:      room.Name,
					Available: existingBooking == 0,
				},
			)
		}

		availableCount := 0

		for _, room := range availability {
			if room.Available {
				availableCount++
			}
		}

		data, err := loadDashboardData(db)

		if err != nil {
			log.Println(
				"Failed to load dashboard data:",
				err,
			)

			http.Error(
				w,
				"Unable to load dashboard data",
				http.StatusInternalServerError,
			)

			return
		}

		data.AvailableRooms = availableCount
		data.CheckIn = checkIn
		data.CheckOut = checkOut
		data.Availability = availability

		if err := tmpl.ExecuteTemplate(
			w,
			"dashboard.html",
			data,
		); err != nil {
			log.Println(
				"Availability template error:",
				err,
			)

			http.Error(
				w,
				"Unable to show availability",
				http.StatusInternalServerError,
			)
		}
	}))

	// =========================================================
	// BOOKING HANDLER
	// =========================================================

	http.HandleFunc("/book", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Unable to process booking",
				http.StatusBadRequest,
			)
			return
		}

		checkIn := r.FormValue("check_in")
		checkOut := r.FormValue("check_out")
		guestsText := r.FormValue("guests")
		roomIDText := r.FormValue("room")

		if checkIn == "" || checkOut == "" {
			http.Error(
				w,
				"Check-in and check-out dates are required",
				http.StatusBadRequest,
			)
			return
		}

		checkInDate, err := time.Parse(
			"2006-01-02",
			checkIn,
		)

		if err != nil {
			http.Error(
				w,
				"Invalid check-in date",
				http.StatusBadRequest,
			)
			return
		}

		checkOutDate, err := time.Parse(
			"2006-01-02",
			checkOut,
		)

		if err != nil {
			http.Error(
				w,
				"Invalid check-out date",
				http.StatusBadRequest,
			)
			return
		}

		if !checkOutDate.After(checkInDate) {
			http.Error(
				w,
				"Check-out date must be after check-in date",
				http.StatusBadRequest,
			)
			return
		}

		guests, err := strconv.Atoi(guestsText)

		if err != nil || guests < 1 {
			http.Error(
				w,
				"Invalid number of guests",
				http.StatusBadRequest,
			)
			return
		}

		roomID, err := strconv.Atoi(roomIDText)

		if err != nil {
			http.Error(
				w,
				"Invalid room",
				http.StatusBadRequest,
			)
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
			http.Error(
				w,
				"Room not found",
				http.StatusBadRequest,
			)
			return
		}

		if guests > selectedRoom.Capacity {
			http.Error(
				w,
				"Too many guests for this room",
				http.StatusBadRequest,
			)
			return
		}

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
			log.Println(
				"Failed to check room availability:",
				err,
			)

			http.Error(
				w,
				"Unable to check room availability",
				http.StatusInternalServerError,
			)

			return
		}

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
				log.Println(
					"Unavailable template error:",
					err,
				)

				http.Error(
					w,
					"Unable to show availability message",
					http.StatusInternalServerError,
				)
			}

			return
		}

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
			log.Println(
				"Failed to save booking:",
				err,
			)

			http.Error(
				w,
				"Unable to save booking",
				http.StatusInternalServerError,
			)

			return
		}

		bookingID, err := result.LastInsertId()

		if err != nil {
			log.Println(
				"Failed to get booking ID:",
				err,
			)

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

		nights := int(checkOutDate.Sub(checkInDate) / (24 * time.Hour))

		total := selectedRoom.Price * nights

		confirmationData := struct {
			ID       int64
			RoomName string
			CheckIn  string
			CheckOut string
			Guests   int
			Price    int
			Nights   int
			Total    int
		}{
			ID:       bookingID,
			RoomName: selectedRoom.Name,
			CheckIn:  checkIn,
			CheckOut: checkOut,
			Guests:   guests,
			Price:    selectedRoom.Price,
			Nights:   nights,
			Total:    total,
		}
		if err := tmpl.ExecuteTemplate(
			w,
			"confirmation.html",
			confirmationData,
		); err != nil {
			log.Println(
				"Confirmation template error:",
				err,
			)

			http.Error(
				w,
				"Unable to show confirmation",
				http.StatusInternalServerError,
			)
		}
	})

	// =========================================================
	// STATIC FILES
	// =========================================================

	http.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	// =========================================================
	// HOMEPAGE
	// =========================================================

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
			log.Println(
				"Template error:",
				err,
			)

			http.Error(
				w,
				"Unable to render page",
				http.StatusInternalServerError,
			)
		}
	})

	log.Println(
		"Hotel platform running at http://localhost:8081",
	)

	log.Fatal(
		http.ListenAndServe(":8081", nil),
	)
}
