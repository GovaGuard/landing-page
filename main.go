package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	contactWorkflowURL = "https://govaguard.app.n8n.cloud/webhook/contact"
)

//go:embed templates/* static/* whitepapers
var content embed.FS

type PageData struct {
	Title string
}

type ContactForm struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"e_mail"`
	Company   string `json:"company"`
	Message   string `json:"message"`
}

func main() {
	// Parse templates
	tmpl := template.Must(template.ParseFS(content, "templates/*.html"))

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Serve whitepaper files
	http.Handle("/whitepapers/", http.FileServer(http.FS(content)))

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

	// Whitepapers page
	http.HandleFunc("/whitepapers", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title: "Whitepapers & Research - GovaGuard",
		}
		tmpl.ExecuteTemplate(w, "whitepapers.html", data)
	})

	// HTMX endpoint for contact form
	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`
					<div class="error-message">
						<div class="error-icon">✗</div>
						<p>Failed to process form. Please try again.</p>
					</div>
				`))
				return
			}

			// Extract form data
			form := ContactForm{
				FirstName: strings.TrimSpace(r.FormValue("first_name")),
				LastName:  strings.TrimSpace(r.FormValue("last_name")),
				Email:     strings.TrimSpace(r.FormValue("email")),
				Company:   strings.TrimSpace(r.FormValue("company")),
				Message:   strings.TrimSpace(r.FormValue("message")),
			}

			// Validate required fields
			if form.FirstName == "" || form.LastName == "" || form.Email == "" || form.Message == "" {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`
					<div class="error-message">
						<div class="error-icon">✗</div>
						<p>Please fill in all required fields.</p>
					</div>
				`))
				return
			}

			var buf bytes.Buffer
			err := json.NewEncoder(&buf).Encode(form)
			if err != nil {
				log.Print(err)
			}

			client := &http.Client{}
			request, err := http.NewRequest(http.MethodPost, contactWorkflowURL, &buf)
			if err != nil {
				log.Print(err)
			}

			request.Header.Set("Content-Type", "application/json")
			request.SetBasicAuth("user", "user")
			response, err := client.Do(request)
			if err != nil {
				log.Print(err)
			}

			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				log.Print(err)
			}

			if response.StatusCode != http.StatusOK {
				log.Print(string(body), response.StatusCode)
			}

			// Success response
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
