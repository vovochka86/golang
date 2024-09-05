package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// PageData represents the data structure passed to the template
type PageData struct {
	Title   string
	Heading string
	Content string
}

// Book represents a book with ID, Title, and Author fields
type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

var tmpl *template.Template

// handler serves the main HTML page
func handler(w http.ResponseWriter, r *http.Request) {
	if err := tmpl.Execute(w, PageData{
		Title:   "LibraGo Library",
		Heading: "Welcome to LibraGo",
		Content: "Your personal library management system.",
	}); err != nil {
		http.Error(w, "Could not execute template", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

// bookHandler serves a list of books as a JSON response
func bookHandler(w http.ResponseWriter, r *http.Request) {
	books := []Book{
		{ID: "1", Title: "Go Programming", Author: "John Doe"},
		// Add more books here
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(books); err != nil {
		http.Error(w, "Failed to encode books data", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %v", err)
	}
}

// fetchBooks fetches books data from a given URL
func fetchBooks(url string) ([]Book, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch books: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body) // Use io.ReadAll instead of ioutil.ReadAll
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var books []Book
	if err := json.Unmarshal(body, &books); err != nil {
		return nil, fmt.Errorf("failed to unmarshal books data: %w", err)
	}

	return books, nil
}

// gracefulShutdown handles server shutdown gracefully
func gracefulShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	log.Println("Shutting down server...")

	// Allow up to 5 seconds for ongoing requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Could not gracefully shut down the server: %v", err)
	}

	log.Println("Server stopped")
}

func main() {
	// Set up logging
	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Parse and cache the template
	tmpl, err = template.ParseFiles("template.html")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handle main page
	http.HandleFunc("/", handler)
	http.HandleFunc("/books", bookHandler)

	// Start the server with graceful shutdown
	server := &http.Server{Addr: ":8080"}
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	gracefulShutdown(server)
}
