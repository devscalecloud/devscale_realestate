package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// User holds a demo account
type User struct {
	Username  string
	Password  string
	FirstName string
	LastName  string
}

// Session holds an authenticated session entry
type Session struct {
	Username  string
	ExpiresAt time.Time
}

// Property represents a house listing for sale or for rent
type Property struct {
	ID          string
	Title       string
	Description string
	Location    string
	Image       string
	Beds        int
	Baths       int
	SqFt        int
	Listing     string // "sale" or "rent"
	Price       float64
	RentPeriod  string // e.g. "month" (only used when Listing == "rent")
}

const demoPassword = "welcome+1"

var users = map[string]User{
	"dapo": {
		Username:  "dapo",
		Password:  demoPassword,
		FirstName: "Dapo",
		LastName:  "",
	},
	"iyiola": {
		Username:  "iyiola",
		Password:  demoPassword,
		FirstName: "Iyiola",
		LastName:  "Olakunle",
	},
	"abayomi": {
		Username:  "abayomi",
		Password:  demoPassword,
		FirstName: "Abayomi",
		LastName:  "Akinyemi",
	},
	"chimezie": {
		Username:  "chimezie",
		Password:  demoPassword,
		FirstName: "Chimezie",
		LastName:  "Ogbu",
	},
}

var properties = []Property{
	{
		ID:          "p1",
		Title:       "Sunset Villa",
		Description: "A bright, modern villa with an open-plan living area, private pool, and landscaped garden — perfect for family living.",
		Location:    "Lekki, Lagos",
		Image:       "https://images.unsplash.com/photo-1613977257363-707ba9348227?w=800&q=80",
		Beds:        4,
		Baths:       3,
		SqFt:        3200,
		Listing:     "sale",
		Price:       450000,
	},
	{
		ID:          "p2",
		Title:       "Maple Court Apartment",
		Description: "A cozy two-bedroom apartment close to shops and transit, with a renovated kitchen and balcony views.",
		Location:    "Ikeja, Lagos",
		Image:       "https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?w=800&q=80",
		Beds:        2,
		Baths:       2,
		SqFt:        1150,
		Listing:     "rent",
		Price:       1800,
		RentPeriod:  "month",
	},
	{
		ID:          "p3",
		Title:       "Riverside Townhouse",
		Description: "Three-story townhouse overlooking the river, featuring a rooftop terrace and attached garage.",
		Location:    "Victoria Island, Lagos",
		Image:       "https://images.unsplash.com/photo-1568605114967-8130f3a36994?w=800&q=80",
		Beds:        3,
		Baths:       3,
		SqFt:        2400,
		Listing:     "sale",
		Price:       620000,
	},
	{
		ID:          "p4",
		Title:       "Downtown Loft",
		Description: "Industrial-style loft with exposed brick, high ceilings, and floor-to-ceiling windows in the heart of downtown.",
		Location:    "Yaba, Lagos",
		Image:       "https://images.unsplash.com/photo-1502672023488-70e25813eb80?w=800&q=80",
		Beds:        1,
		Baths:       1,
		SqFt:        850,
		Listing:     "rent",
		Price:       1200,
		RentPeriod:  "month",
	},
	{
		ID:          "p5",
		Title:       "Golden Acres Estate",
		Description: "A sprawling estate on 2 acres of land with a private garden, guest house, and four-car garage.",
		Location:    "Ikoyi, Lagos",
		Image:       "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?w=800&q=80",
		Beds:        5,
		Baths:       5,
		SqFt:        5400,
		Listing:     "sale",
		Price:       1250000,
	},
	{
		ID:          "p6",
		Title:       "Harbor View Studio",
		Description: "A compact, stylish studio apartment with harbor views, ideal for a single professional or a couple.",
		Location:    "Ajah, Lagos",
		Image:       "https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?w=800&q=80",
		Beds:        1,
		Baths:       1,
		SqFt:        600,
		Listing:     "rent",
		Price:       850,
		RentPeriod:  "month",
	},
	{
		ID:          "p7",
		Title:       "Cedar Grove Bungalow",
		Description: "Single-story bungalow with a spacious backyard, updated fixtures, and a quiet, tree-lined street.",
		Location:    "Surulere, Lagos",
		Image:       "https://images.unsplash.com/photo-1570129477492-45c003edd2be?w=800&q=80",
		Beds:        3,
		Baths:       2,
		SqFt:        1800,
		Listing:     "sale",
		Price:       310000,
	},
	{
		ID:          "p8",
		Title:       "Palm Heights Duplex",
		Description: "Modern duplex with a private entrance, dedicated parking, and access to a shared rooftop lounge.",
		Location:    "Magodo, Lagos",
		Image:       "https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?w=800&q=80",
		Beds:        3,
		Baths:       2,
		SqFt:        2000,
		Listing:     "rent",
		Price:       2200,
		RentPeriod:  "month",
	},
}

var (
	sessions = map[string]Session{}
	mu       sync.RWMutex
	tmpl     *template.Template

	appLog *log.Logger
)

func propertyByID(id string) (Property, bool) {
	for _, p := range properties {
		if p.ID == id {
			return p, true
		}
	}
	return Property{}, false
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return r.RemoteAddr
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// formatMoney formats a float64 as a US dollar string e.g. $450,000.00
func formatMoney(amount float64) string {
	s := fmt.Sprintf("%.2f", amount)
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	decimals := parts[1]

	var result []byte
	n := len(intPart)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, intPart[i])
	}
	return "$" + string(result) + "." + decimals
}

func initials(firstName, lastName string) string {
	var b strings.Builder
	if len(firstName) > 0 {
		b.WriteString(strings.ToUpper(firstName[:1]))
	}
	if len(lastName) > 0 {
		b.WriteString(strings.ToUpper(lastName[:1]))
	}
	return b.String()
}

func fullName(firstName, lastName string) string {
	if lastName == "" {
		return firstName
	}
	return firstName + " " + lastName
}

func getSession(r *http.Request) (User, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return User{}, false
	}
	mu.RLock()
	session, ok := sessions[cookie.Value]
	mu.RUnlock()
	if !ok || time.Now().After(session.ExpiresAt) {
		return User{}, false
	}
	user, exists := users[session.Username]
	return user, exists
}

// loginHandler handles GET / (render login) and POST / (authenticate)
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if _, ok := getSession(r); ok {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "login.html", nil); err != nil {
			log.Printf("Login template error: %v", err)
		}
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	mu.RLock()
	user, ok := users[username]
	mu.RUnlock()

	if !ok || subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) != 1 {
		appLog.Printf("LOGIN_FAILED username=%q ip=%s", username, clientIP(r))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Error": "Invalid username or password. Please try again.",
		})
		return
	}

	token, err := generateToken()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	sessions[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	appLog.Printf("LOGIN_SUCCESS username=%q ip=%s", username, clientIP(r))

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// registerHandler handles GET /register (render form) and POST /register (create account)
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := getSession(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "register.html", nil); err != nil {
			log.Printf("Register template error: %v", err)
		}
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")
	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))

	renderErr := func(msg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
			"Error": msg,
		})
	}

	if username == "" || password == "" || firstName == "" || lastName == "" {
		renderErr("All fields are required.")
		return
	}
	if len(username) < 3 {
		renderErr("Username must be at least 3 characters long.")
		return
	}
	if len(password) < 6 {
		renderErr("Password must be at least 6 characters long.")
		return
	}
	if password != confirmPassword {
		renderErr("Passwords do not match.")
		return
	}

	mu.RLock()
	_, exists := users[username]
	mu.RUnlock()
	if exists {
		renderErr("Username already exists. Please choose a different username.")
		return
	}

	mu.Lock()
	users[username] = User{
		Username:  username,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
	}
	mu.Unlock()

	appLog.Printf("REGISTER username=%q ip=%s", username, clientIP(r))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
		"Success": true,
	})
}

// dashboardHandler renders the property listing dashboard
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getSession(r)
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	type PropertyView struct {
		Property
		PriceFormatted string
	}

	var listings []PropertyView
	for _, p := range properties {
		listings = append(listings, PropertyView{
			Property:       p,
			PriceFormatted: formatMoney(p.Price),
		})
	}

	data := map[string]interface{}{
		"User":       user,
		"Properties": listings,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("Dashboard template error: %v", err)
	}
}

// buyHandler processes a mock purchase (POST /buy)
func buyHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getSession(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	propertyID := r.FormValue("property_id")
	property, ok := propertyByID(propertyID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	appLog.Printf("BUY username=%q property=%q title=%q amount=%.2f ip=%s",
		user.Username, property.ID, property.Title, property.Price, clientIP(r))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, `{"success": true, "message": "%s has been bought."}`, template.JSEscapeString(property.Title))
}

// rentHandler processes a mock rental payment (POST /rent)
func rentHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getSession(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	propertyID := r.FormValue("property_id")
	property, ok := propertyByID(propertyID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	appLog.Printf("RENT username=%q property=%q title=%q amount=%.2f ip=%s",
		user.Username, property.ID, property.Title, property.Price, clientIP(r))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, `{"success": true, "message": "%s has been rented successfully."}`, template.JSEscapeString(property.Title))
}

// logoutHandler destroys the session and redirects to login
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		mu.Lock()
		session, ok := sessions[cookie.Value]
		delete(sessions, cookie.Value)
		mu.Unlock()
		if ok {
			appLog.Printf("LOGOUT username=%q ip=%s", session.Username, clientIP(r))
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	appLog = log.New(&logWriter{}, "", log.Ldate|log.Ltime)

	var err error
	tmpl, err = template.New("").Funcs(template.FuncMap{
		"initials": initials,
		"fullName": fullName,
	}).ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v\nMake sure you run this from the project root directory.", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", loginHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/dashboard", dashboardHandler)
	mux.HandleFunc("/buy", buyHandler)
	mux.HandleFunc("/rent", rentHandler)
	mux.HandleFunc("/logout", logoutHandler)

	port := ":8080"
	fmt.Println("============================================")
	fmt.Println("  DevScale Real Estate")
	fmt.Println("============================================")
	fmt.Printf("  Server running at http://0.0.0.0%s\n", port)
	fmt.Println("  Open your browser: http://localhost:8080")
	fmt.Println("  Demo login: username is <your username> and password is welcome+1")
	fmt.Println("============================================")
	log.Fatal(http.ListenAndServe(port, mux))
}

// logWriter writes app log lines to stdout
type logWriter struct{}

func (w *logWriter) Write(p []byte) (int, error) {
	return fmt.Print(string(p))
}
