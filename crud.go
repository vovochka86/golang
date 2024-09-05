package main

import (
    "context"
    "encoding/json"
    
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"

    "github.com/gorilla/mux"
)

// Book structure
type Book struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Author string `json:"author"`
}

var books []Book
var nextID int

// Create a new book (POST /books)
func createBookHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("Received request: Create a new book")
    r.ParseForm()
    title := r.FormValue("title")
    author := r.FormValue("author")

    nextID++
    book := Book{
        ID:     strconv.Itoa(nextID),
        Title:  title,
        Author: author,
    }
    books = append(books, book)

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(book)

    log.Printf("Created a new book: %+v\n", book)
}

// Update an existing book (PUT /books/{id})
func updateBookHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("Received request: Update a book")
    vars := mux.Vars(r)
    id := vars["id"]

    var updatedBook Book
    if err := json.NewDecoder(r.Body).Decode(&updatedBook); err != nil {
        http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
        log.Printf("Error decoding JSON: %v\n", err)
        return
    }

    for i, book := range books {
        if book.ID == id {
            books[i] = updatedBook
            books[i].ID = id
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(books[i])

            log.Printf("Updated book ID %s: %+v\n", id, books[i])
            return
        }
    }

    http.Error(w, "Book not found", http.StatusNotFound)
    log.Printf("Book not found: ID %s\n", id)
}

// Delete a book (DELETE /books/{id})
func deleteBookHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("Received request: Delete a book")
    vars := mux.Vars(r)
    id := vars["id"]

    for i, book := range books {
        if book.ID == id {
            books = append(books[:i], books[i+1:]...)
            w.WriteHeader(http.StatusNoContent)
            log.Printf("Deleted book ID %s\n", id)
            return
        }
    }

    http.Error(w, "Book not found", http.StatusNotFound)
    log.Printf("Book not found: ID %s\n", id)
}

// List all books (GET /books)
func listBooksHandler(w http.ResponseWriter, r *http.Request) {
    log.Println("Received request: List all books")
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(books)
}

// Serve HTML page (GET /)
func serveHTML(w http.ResponseWriter, r *http.Request) {
    log.Println("Received request: Serve HTML page")
    http.ServeFile(w, r, "template.html")
}

// Graceful shutdown and routes setup
func main() {
    r := mux.NewRouter()

    // Set up routes
    r.HandleFunc("/", serveHTML).Methods(http.MethodGet)
    r.HandleFunc("/books", createBookHandler).Methods(http.MethodPost)
    r.HandleFunc("/books/{id}", updateBookHandler).Methods(http.MethodPut)
    r.HandleFunc("/books/{id}", deleteBookHandler).Methods(http.MethodDelete)
    r.HandleFunc("/books", listBooksHandler).Methods(http.MethodGet)

    srv := &http.Server{
        Addr:    ":8080",
        Handler: r,
    }

    // Graceful shutdown channel
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        log.Println("Server started at :8080")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Could not listen on :8080: %v\n", err)
        }
    }()

    <-quit
    log.Println("Shutting down server...")

    // Graceful shutdown with timeout context
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v\n", err)
    }

    log.Println("Server exiting")
}
