package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed templates/* static/*
var content embed.FS

type PageData struct {
	Title string
}

func main() {
	// Parse templates
	tmpl := template.Must(template.ParseFS(content, "templates/*.html"))

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Landing page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title: "GovaGuard — CTO & CISO Consulting",
		}
		tmpl.ExecuteTemplate(w, "index.html", data)
	})

	// Imprint page
	http.HandleFunc("/imprint", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title: "Imprint - GovaGuard",
		}
		tmpl.ExecuteTemplate(w, "imprint.html", data)
	})

	// HTMX endpoint for contact form
	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			// In production, process form data here
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<div class="success-message">
					<div class="success-icon">✓</div>
					<p>Message received. We'll respond within 24 hours.</p>
				</div>
			`))
		}
	})

	log.Println("GovaGuard landing page running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
