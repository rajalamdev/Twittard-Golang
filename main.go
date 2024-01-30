// main.go

package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

// Buat session store baru dengan kunci rahasia
var store = sessions.NewCookieStore([]byte("your-secret-key"))

// Struct User merepresentasikan tabel user dalam database
type User struct {
	ID       int
	Username string
	Password string
}

// Struct Tweet merepresentasikan tabel tweet dalam database
type Tweet struct {
	ID        int
	UserID    int
	Text      string
	Username  string // Tambahkan field ini untuk menyimpan username yang terkait dengan tweet
	CreatedAt string
}

func main() {
	// Tentukan rute HTTP
	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/loginProcess", loginProcess)
	http.HandleFunc("/home", isAuthenticated(home))
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/addTweet", isAuthenticated(addTweet))
	http.HandleFunc("/addTweetProcess", isAuthenticated(addTweetProcess))
	http.HandleFunc("/myTweet", isAuthenticated(myTweet))
	http.HandleFunc("/deleteTweet", isAuthenticated(deleteTweet))
	http.HandleFunc("/add_user", add_user)
	http.HandleFunc("/addUserProcess", addUserProcess)

	// Layani file statis
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Mulai server pada port 8080
	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

// index mengatasi endpoint root dan merender template login
func index(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login", nil)
}

// login mengatasi endpoint login dan merender template login
func login(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login", nil)
}

// loginProcess mengatasi pengiriman formulir login
func loginProcess(w http.ResponseWriter, r *http.Request) {
	// Periksa jika metode permintaan adalah POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Ambil username dan password dari formulir
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Hubungkan ke database
	db := dbConn()
	defer db.Close()

	// Query tabel user untuk username dan password yang diberikan
	row := db.QueryRow("SELECT id, username, password FROM users WHERE username = ? AND password = ?", username, password)

	var user User
	// Pindai hasil ke dalam struktur user
	err := row.Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		// Jika login gagal, render template login dengan pesan kesalahan
		log.Println("Login failed:", err)
		renderTemplate(w, "login", map[string]interface{}{"Message": "Username atau password salah."})
		return
	}

	// Dapatkan sesi dan simpan ID pengguna
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["userID"] = user.ID
	session.Save(r, w)

	// Redirect ke halaman beranda
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

// Fungsi untuk menampilkan halaman home
func home(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mendapatkan ID pengguna dari session
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Dapatkan informasi pengguna
	currentUser, err := getUserInfo(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Dapatkan tweet dari database
	tweets, err := getTweets()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Gabungkan data pengguna dan tweet
	data := map[string]interface{}{
		"CurrentUser": currentUser,
		"Tweets":      tweets,
	}

	// Render template
	renderTemplate(w, "home", data)
}

// Fungsi untuk mendapatkan informasi pengguna berdasarkan ID
func getUserInfo(userID int) (User, error) {
	db := dbConn()
	defer db.Close()

	// Query untuk mendapatkan informasi pengguna berdasarkan ID
	row := db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID)

	var user User
	err := row.Scan(&user.ID, &user.Username)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// Fungsi untuk menampilkan halaman "My Tweet"
func myTweet(w http.ResponseWriter, r *http.Request) {
	// Mendapatkan informasi pengguna dari session
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mendapatkan ID pengguna dari session
	userID, ok := session.Values["userID"].(int)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Dapatkan tweet pengguna dari database
	userTweets, err := getUserTweets(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Gabungkan data pengguna dan tweet
	data := map[string]interface{}{
		"UserTweets": userTweets,
	}

	// Render template "my_tweet.html" dengan data
	renderTemplate(w, "my_tweet", data)
}

// Fungsi untuk delete tweet
func deleteTweet(w http.ResponseWriter, r *http.Request) {
	// Pastikan metodenya adalah POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Ambil nilai tweetID dari form
	tweetID := r.FormValue("tweetID")

	// Hapus tweet dari database
	db := dbConn()
	delForm, err := db.Prepare("DELETE FROM tweet WHERE id = ?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = delForm.Exec(tweetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	// Redirect ke halaman "My Tweet" setelah menghapus tweet
	http.Redirect(w, r, "/myTweet", http.StatusSeeOther)
}

// Fungsi untuk mendapatkan tweet pengguna berdasarkan ID pengguna
func getUserTweets(userID int) ([]Tweet, error) {
	// Hubungkan ke database
	db := dbConn()
	defer db.Close()

	// Query untuk mendapatkan tweet pengguna berdasarkan ID pengguna
	selDB, err := db.Query("SELECT id, tweet_text, createdAt FROM tweet WHERE userid = ? ORDER BY id DESC", userID)
	if err != nil {
		return nil, err
	}
	defer selDB.Close()

	var userTweets []Tweet
	// Pindai hasil ke dalam struktur Tweet
	for selDB.Next() {
		var tweet Tweet
		err := selDB.Scan(&tweet.ID, &tweet.Text, &tweet.CreatedAt)
		if err != nil {
			return nil, err
		}

		userTweets = append(userTweets, tweet)
	}

	return userTweets, nil
}
func add_user(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "add_user", nil)
}

func addUserProcess(w http.ResponseWriter, r *http.Request) {
	// Pastikan metodenya adalah POST
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Retrieve values from the form
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate input (you can add more validation as needed)
	if username == "" || password == "" {
		// Handle invalid input
		http.Error(w, "Invalid username or password", http.StatusBadRequest)
		return
	}

	// Connect to the database
	db := dbConn()
	defer db.Close()

	// Check if the username is already taken
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		log.Println("Error checking username existence:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		// Username is already taken, handle appropriately (e.g., redirect with error message)
		log.Println("Username already taken:", username)
		http.Redirect(w, r, "/add_user?error=username_taken", http.StatusSeeOther)
		return
	}

	// Prepare the INSERT statement for adding a new user
	insForm, err := db.Prepare("INSERT INTO users(username, password) VALUES(?, ?)")
	if err != nil {
		log.Println("Error preparing INSERT statement:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the statement to insert a new user into the database with the plaintext password
	_, err = insForm.Exec(username, password)
	if err != nil {
		log.Println("Error executing INSERT statement:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the home page after adding the user
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}


// Fungsi untuk mendapatkan tweet dari database
func getTweets() ([]Tweet, error) {
	// Hubungkan ke database
	db := dbConn()
	defer db.Close()

	// Query untuk menggabungkan tabel tweet dan user untuk mendapatkan tweet dengan username
	selDB, err := db.Query("SELECT tweet.id, tweet.userid, tweet.tweet_text, tweet.createdAt, users.username FROM tweet JOIN users ON tweet.userid = users.id ORDER BY tweet.id DESC")
	if err != nil {
		return nil, err
	}
	defer selDB.Close()

	var tweets []Tweet
	// Pindai hasil ke dalam struktur Tweet
	for selDB.Next() {
		var tweet Tweet
		err := selDB.Scan(&tweet.ID, &tweet.UserID, &tweet.Text, &tweet.CreatedAt, &tweet.Username)
		if err != nil {
			return nil, err
		}

		tweets = append(tweets, tweet)
	}

	return tweets, nil
}

// Fungsi untuk menampilkan halaman tambah tweet
func addTweet(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "add_tweet", nil)
}

// Fungsi untuk menangani proses tambah tweet
func addTweetProcess(w http.ResponseWriter, r *http.Request) {
	// Pastikan metodenya adalah POST
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Ambil nilai dari form
	text := r.FormValue("text")

	// Simpan tweet ke database
	db := dbConn()
	insForm, err := db.Prepare("INSERT INTO tweet(userid, tweet_text) VALUES(?, ?)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ganti userID dengan sesuai dengan user yang sedang login
	userID := getCurrentUserID(r)
	_, err = insForm.Exec(userID, text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	// Redirect ke halaman utama setelah menambahkan tweet
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

// Fungsi untuk mendapatkan ID pengguna yang sedang login dari session
func getCurrentUserID(r *http.Request) int {
	session, err := store.Get(r, "session-name")
	if err != nil {
		return 0
	}

	userID, ok := session.Values["userID"].(int)
	if !ok {
		return 0
	}

	return userID
}

// logout mengatasi endpoint logout dan menghapus sesi pengguna
func logout(w http.ResponseWriter, r *http.Request) {
	// Dapatkan sesi dan hapus ID pengguna
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	delete(session.Values, "userID")
	session.Save(r, w)

	// Redirect ke halaman login
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// renderTemplate merender template HTML dengan data
func renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	// Parse file template
	tmplFile := "templates/" + tmplName + ".html"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Eksekusi template dengan data
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// isAuthenticated adalah middleware untuk memeriksa otentikasi sebelum menjalankan fungsi berikutnya
func isAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dapatkan sesi
		session, err := store.Get(r, "session-name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Periksa apakah ada userID dalam sesi
		if _, ok := session.Values["userID"].(int); ok {
			// Jika ada, jalankan fungsi berikutnya
			next.ServeHTTP(w, r)
		} else {
			// Jika tidak, redirect ke halaman login
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})
}

// dbConn menghubungkan ke database MySQL
func dbConn() (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := "root"
	dbPass := ""
	dbName := "twittard"
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Server is runnin!")
	return db
}
