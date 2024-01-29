// main.go

package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"github.com/gorilla/sessions"
	_ "github.com/go-sql-driver/mysql"
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
	ID       int
	UserID   int
	Text     string
	Username string // Tambahkan field ini untuk menyimpan username yang terkait dengan tweet
}

func main() {
	// Tentukan rute HTTP
	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/loginProcess", loginProcess)
	http.HandleFunc("/home", isAuthenticated(home))
	http.HandleFunc("/logout", logout)

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

// home menampilkan halaman beranda dengan tweet
func home(w http.ResponseWriter, r *http.Request) {
	// Dapatkan tweet dari database
	tweets, err := getTweets()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render template home dengan data tweet
	renderTemplate(w, "home", map[string]interface{}{"Tweets": tweets})
}

// getTweets mengambil tweet dari database
func getTweets() ([]Tweet, error) {
	// Hubungkan ke database
	db := dbConn()
	defer db.Close()

	// Query untuk menggabungkan tabel tweet dan user untuk mendapatkan tweet dengan username
	selDB, err := db.Query("SELECT tweet.id, tweet.userid, tweet.tweet_text, users.username FROM tweet JOIN users ON tweet.userid = users.id ORDER BY tweet.id DESC")
	if err != nil {
		return nil, err
	}
	defer selDB.Close()

	var tweets []Tweet
	// Pindai hasil ke dalam struktur Tweet
	for selDB.Next() {
		var tweet Tweet
		err := selDB.Scan(&tweet.ID, &tweet.UserID, &tweet.Text, &tweet.Username)
		if err != nil {
			return nil, err
		}
		tweets = append(tweets, tweet)
	}

	return tweets, nil
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
