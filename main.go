package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/mux"
    "github.com/gorilla/sessions"
    "github.com/gorilla/websocket"
    "github.com/shirou/gopsutil/cpu"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/mem"
)

var (
    store = sessions.NewCookieStore([]byte("very-secret-key"))
    upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }
    clients      = make(map[*websocket.Conn]string)
    clientsMutex sync.Mutex
    messages     []Message
    msgIDCounter = 0
    templates    = template.Must(template.ParseGlob("templates/*.html"))
)

type Message struct {
    ID      int       `json:"id"`
    User    string    `json:"user"`
    Content string    `json:"content"`
    Time    time.Time `json:"time"`
}

var users = map[string]string{}

const usersFile = "users.json"

func init() {
    if _, err := os.Stat(usersFile); os.IsNotExist(err) {
        fmt.Println("First run detected.")
        fmt.Println("The default admin username is 'admin' and the password is 'server'.")
        fmt.Print("How many regular users do you want to create? ")
        var n int
        fmt.Scanln(&n)

        for i := 0; i < n; i++ {
            var username, password string
            fmt.Printf("Enter username #%d: ", i+1)
            fmt.Scanln(&username)
            fmt.Printf("Enter password for %s: ", username)
            fmt.Scanln(&password)
            users[username] = password
        }

        users["admin"] = "server"

        f, _ := os.Create(usersFile)
        json.NewEncoder(f).Encode(users)
        f.Close()
        fmt.Println("✅ Users saved to users.json")
    } else {
        f, _ := os.Open(usersFile)
        json.NewDecoder(f).Decode(&users)
        f.Close()
    }
}

func main() {
    os.MkdirAll("uploads", 0755)

    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   86400 * 7,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
        Secure:   false,
    }

    r := mux.NewRouter()

    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

    r.HandleFunc("/", loginHandler).Methods("GET", "POST")
    r.HandleFunc("/logout", logoutHandler).Methods("GET")
    r.HandleFunc("/home", authMiddleware(homeHandler)).Methods("GET")
    r.HandleFunc("/uploads", authMiddleware(uploadsPage)).Methods("GET")
    r.HandleFunc("/stats", authMiddleware(statsPage)).Methods("GET")
    r.HandleFunc("/settings", authMiddleware(settingsPage)).Methods("GET")
    r.HandleFunc("/chat", authMiddleware(chatPage)).Methods("GET")
    r.HandleFunc("/admin", authMiddleware(adminPage)).Methods("GET")

    r.HandleFunc("/api/upload", authMiddleware(uploadHandler)).Methods("POST")
    r.HandleFunc("/api/listImages", authMiddleware(listImagesHandler)).Methods("GET")
    r.HandleFunc("/api/userStats", authMiddleware(userStatsHandler)).Methods("GET")
    r.HandleFunc("/api/systemStats", authMiddleware(systemStatsHandler)).Methods("GET")
    r.HandleFunc("/api/deleteImage", authMiddleware(deleteImageHandler)).Methods("POST")
    r.HandleFunc("/api/listMessages", authMiddleware(listMessagesHandler)).Methods("GET")
    r.HandleFunc("/api/deleteMessage", authMiddleware(deleteMessageHandler)).Methods("POST")

    r.HandleFunc("/ws", authMiddleware(chatWSHandler))

    fmt.Println("Listening on http://127.0.0.1:3000")
    log.Fatal(http.ListenAndServe(":3000", r))
}

// ===== PAGE HANDLERS =====

func loginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        u, p := r.FormValue("username"), r.FormValue("password")
        if correctPwd, ok := users[u]; ok && p == correctPwd {
            sess, _ := store.Get(r, "session")
            sess.Values["user"] = u
            sess.Save(r, w)
            http.Redirect(w, r, "/home", http.StatusFound)
            return
        }
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    templates.ExecuteTemplate(w, "login.html", nil)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
    sess, _ := store.Get(r, "session")
    delete(sess.Values, "user")
    sess.Save(r, w)
    http.Redirect(w, r, "/", http.StatusFound)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    user := getSessionUser(r)
    templates.ExecuteTemplate(w, "home.html", map[string]string{"User": user})
}

func uploadsPage(w http.ResponseWriter, r *http.Request) {
    user := getSessionUser(r)
    templates.ExecuteTemplate(w, "uploads.html", map[string]string{"User": user})
}

func statsPage(w http.ResponseWriter, r *http.Request) {
    user := getSessionUser(r)
    templates.ExecuteTemplate(w, "stats.html", map[string]string{"User": user})
}

func settingsPage(w http.ResponseWriter, r *http.Request) {
    user := getSessionUser(r)
    templates.ExecuteTemplate(w, "settings.html", map[string]string{"User": user})
}

func chatPage(w http.ResponseWriter, r *http.Request) {
    user := getSessionUser(r)
    templates.ExecuteTemplate(w, "chat.html", map[string]string{"User": user})
}

func adminPage(w http.ResponseWriter, r *http.Request) {
    user := getSessionUser(r)
    if user != "admin" {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    templates.ExecuteTemplate(w, "admin.html", map[string]string{"User": user})
}

// ===== API HANDLERS =====

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    f, hdr, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "No file uploaded", http.StatusBadRequest)
        return
    }
    defer f.Close()

    dst, err := os.Create(filepath.Join("uploads", hdr.Filename))
    if err != nil {
        http.Error(w, "Cannot save file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()
    io.Copy(dst, f)
    w.Write([]byte("ok"))
}

func listImagesHandler(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query().Get("q")
    files, _ := os.ReadDir("uploads")
    var out []string
    for _, fi := range files {
        if !fi.IsDir() {
            name := fi.Name()
            if q == "" || strings.Contains(strings.ToLower(name), strings.ToLower(q)) {
                out = append(out, name)
            }
        }
    }
    json.NewEncoder(w).Encode(out)
}

func userStatsHandler(w http.ResponseWriter, r *http.Request) {
    files, _ := os.ReadDir("uploads")
    uploadCount := len(files)
    msgCount := 0
    user := getSessionUser(r)
    for _, m := range messages {
        if m.User == user {
            msgCount++
        }
    }
    json.NewEncoder(w).Encode(map[string]int{"uploads": uploadCount, "messages": msgCount})
}

func systemStatsHandler(w http.ResponseWriter, r *http.Request) {
    cpuPerc, _ := cpu.Percent(0, false)
    vm, _ := mem.VirtualMemory()
    ds, _ := disk.Usage("/")
    stats := map[string]interface{}{
        "CPU":         fmt.Sprintf("%.1f%%", cpuPerc[0]),
        "MemoryUsed":  vm.Used,
        "MemoryTotal": vm.Total,
        "DiskUsed":    ds.Used,
        "DiskTotal":   ds.Total,
    }
    json.NewEncoder(w).Encode(stats)
}

func deleteImageHandler(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    img := r.FormValue("image")

    // Clean the provided path
    cleanImage := filepath.Clean(img)

    // Define and normalize the base uploads directory
    baseDir := filepath.Clean("uploads")
    fullPath := filepath.Join(baseDir, cleanImage)

    // Ensure the path stays within the uploads directory
    if !strings.HasPrefix(fullPath, baseDir+string(os.PathSeparator)) {
        http.Error(w, "Invalid file path", http.StatusBadRequest)
        return
    }

    // Attempt to delete the file
    if err := os.Remove(fullPath); err != nil {
        http.Error(w, "Failed to delete file", http.StatusInternalServerError)
        return
    }

    w.Write([]byte("deleted"))
}

func listMessagesHandler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(messages)
}

func deleteMessageHandler(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    id, _ := strconv.Atoi(r.FormValue("id"))
    for i, m := range messages {
        if m.ID == id {
            messages = append(messages[:i], messages[i+1:]...)
            break
        }
    }
    w.Write([]byte("deleted"))
}

// ===== WebSocket Chat =====

func chatWSHandler(w http.ResponseWriter, r *http.Request) {
    sess, _ := store.Get(r, "session")
    user, _ := sess.Values["user"].(string)

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    clientsMutex.Lock()
    clients[conn] = user
    clientsMutex.Unlock()

    for _, msg := range messages {
        conn.WriteJSON(msg)
    }

    for {
        var inc struct{ Content string }
        if err := conn.ReadJSON(&inc); err != nil {
            break
        }
        msg := Message{
            ID:      msgIDCounter,
            User:    user,
            Content: inc.Content,
            Time:    time.Now(),
        }
        msgIDCounter++
        messages = append(messages, msg)

        clientsMutex.Lock()
        for c := range clients {
            c.WriteJSON(msg)
        }
        clientsMutex.Unlock()
    }

    clientsMutex.Lock()
    delete(clients, conn)
    clientsMutex.Unlock()
    conn.Close()
}

// ===== MIDDLEWARE =====

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sess, _ := store.Get(r, "session")
        user, ok := sess.Values["user"].(string)
        if !ok || user == "" {
            http.Redirect(w, r, "/", http.StatusFound)
            return
        }
        next(w, r)
    }
}

func getSessionUser(r *http.Request) string {
    sess, _ := store.Get(r, "session")
    if u, ok := sess.Values["user"].(string); ok {
        return u
    }
    return ""
}
