package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
)

const (
	defaultNotionAPIURL = "https://api.notion.com/v1/pages"
	notionVersion       = "2022-06-28"
)

//go:embed templates/* static/*
var content embed.FS

type PageData struct {
	Title       string
	Description string
}

type ContactForm struct {
	Name       string
	Email      string
	Company    string
	Message    string
	Framework  string
	SourcePage string
}

// frameworkOptions maps form select values to Notion "Framework" select names.
var frameworkOptions = map[string]string{
	"iso-27001": "ISO 27001",
	"c5":        "BSI C5",
	"soc-2":     "SOC 2",
	"nis2":      "NIS2",
	"other":     "Other / not sure",
}

// notionPage builds the Notion "create page" payload for a contact submission.
// The target database needs these properties: Name (title), Email (email),
// Company (rich text), Message (rich text), Status (select with a "New" option).
func notionPage(databaseID string, form ContactForm) map[string]any {
	// Notion caps a single rich_text element at 2000 characters; split longer
	// values (the message can be up to 5000) into multiple elements.
	richText := func(s string) []map[string]any {
		const maxRunes = 2000
		runes := []rune(s)
		out := []map[string]any{}
		for start := 0; ; start += maxRunes {
			end := min(start+maxRunes, len(runes))
			out = append(out, map[string]any{"text": map[string]any{"content": string(runes[start:end])}})
			if end == len(runes) {
				break
			}
		}
		return out
	}
	properties := map[string]any{
		"Name":    map[string]any{"title": richText(form.Name)},
		"Email":   map[string]any{"email": form.Email},
		"Company": map[string]any{"rich_text": richText(form.Company)},
		"Message": map[string]any{"rich_text": richText(form.Message)},
		"Source":  map[string]any{"rich_text": richText(form.SourcePage)},
		"Status":  map[string]any{"select": map[string]any{"name": "New"}},
	}
	if name, ok := frameworkOptions[form.Framework]; ok {
		properties["Framework"] = map[string]any{"select": map[string]any{"name": name}}
	}
	return map[string]any{
		"parent":     map[string]any{"database_id": databaseID},
		"properties": properties,
	}
}

const successFragment = `
	<div class="success-message">
		<div class="success-icon">✓</div>
		<p>Message received. We'll respond within 24 hours.</p>
	</div>
`

func errorFragment(msg string) string {
	return `
		<div class="error-message">
			<div class="error-icon">✗</div>
			<p>` + template.HTMLEscapeString(msg) + `</p>
		</div>
	`
}

func main() {
	// Parse templates
	tmpl := template.Must(template.ParseFS(content, "templates/*.html"))

	notionAPIURL := os.Getenv("NOTION_API_URL")
	if notionAPIURL == "" {
		notionAPIURL = defaultNotionAPIURL
	}
	notionAPIKey := os.Getenv("NOTION_API_KEY")
	notionDatabaseID := os.Getenv("NOTION_DATABASE_ID")
	if notionAPIKey == "" || notionDatabaseID == "" {
		log.Println("WARNING: NOTION_API_KEY/NOTION_DATABASE_ID not set; contact form submissions will fail")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Landing page (and 404 for unknown paths, since "/" is a catch-all)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := PageData{
			Title:       "GovaGuard — ISO 27001, C5 & SOC 2 Consulting from Germany",
			Description: "CTO & CISO consulting for security certifications: ISO 27001 and BSI C5 in Europe, SOC 2 and HIPAA for the US market. Executive-led, fixed scope.",
		}
		tmpl.ExecuteTemplate(w, "index.html", data)
	})

	// Framework pages
	http.HandleFunc("/iso-27001", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "ISO 27001 Consulting & Certification Readiness | GovaGuard",
			Description: "From gap analysis to passed audit: executive-led ISO 27001 consulting for startups and scale-ups. Realistic timelines, no bloated ISMS theater.",
		}
		tmpl.ExecuteTemplate(w, "iso27001.html", data)
	})

	http.HandleFunc("/soc-2", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "SOC 2 Readiness Consulting for EU Companies | GovaGuard",
			Description: "Your US enterprise deal is waiting on a SOC 2 report. We take EU SaaS teams from zero to auditor-ready, reusing your ISO 27001 work instead of duplicating it.",
		}
		tmpl.ExecuteTemplate(w, "soc2.html", data)
	})

	http.HandleFunc("/compare/soc-2-vs-iso-27001", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "SOC 2 vs ISO 27001 — Which Do You Need? | GovaGuard",
			Description: "An honest comparison from a consultancy that runs both programs: who asks for which, what they cost, and when one control set can produce both.",
		}
		tmpl.ExecuteTemplate(w, "soc2-vs-iso27001.html", data)
	})

	http.HandleFunc("/de/c5-testat", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "BSI C5 Testat Beratung — Typ 1 & Typ 2 | GovaGuard",
			Description: "C5-Testat für Cloud-Anbieter: Kriterienkatalog-Mapping, Nachweisführung und Begleitung bis zum Testat — inkl. § 393 SGB V für das Gesundheitswesen.",
		}
		tmpl.ExecuteTemplate(w, "c5-testat.html", data)
	})

	http.HandleFunc("/dora", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "DORA Compliance for ICT Providers & Banks | GovaGuard",
			Description: "In force since January 2025: we bring financial entities and their ICT providers up to DORA's operational-resilience bar — contracts, testing, reporting.",
		}
		tmpl.ExecuteTemplate(w, "dora.html", data)
	})

	http.HandleFunc("/hipaa", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "HIPAA Compliance & HITRUST Readiness for Health Tech | GovaGuard",
			Description: "Selling digital health into the US? HIPAA is the entry ticket, HITRUST is what hospital procurement asks for. Executive-led readiness for both — SaMD experience included.",
		}
		tmpl.ExecuteTemplate(w, "hipaa.html", data)
	})

	http.HandleFunc("/iso-42001", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "ISO 42001 Consulting — AI Management Systems | GovaGuard",
			Description: "Enterprise buyers are starting to ask how you govern AI. ISO 42001 is the certifiable answer — and it bolts onto an existing ISO 27001 ISMS instead of starting from zero.",
		}
		tmpl.ExecuteTemplate(w, "iso42001.html", data)
	})

	http.HandleFunc("/cra", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "Cyber Resilience Act Readiness & CE Marking | GovaGuard",
			Description: "ENISA reporting is live since September 2026; CE marking becomes mandatory in December 2027. CRA readiness for software and connected products in the EU.",
		}
		tmpl.ExecuteTemplate(w, "cra.html", data)
	})

	http.HandleFunc("/de/c5-vs-iso-27001", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "C5 oder ISO 27001? Der ehrliche Vergleich | GovaGuard",
			Description: "Testat oder Zertifikat? Wer was fordert, was beides kostet und warum die Kombination auf einem Kontrollset meist gewinnt — von Beratern, die beide Wege begleiten.",
		}
		tmpl.ExecuteTemplate(w, "c5-vs-iso27001.html", data)
	})

	http.HandleFunc("/de/iso-27001-beratung", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "ISO 27001 Beratung: Zertifizierung ohne Umwege | GovaGuard",
			Description: "Von der Gap-Analyse bis zum bestandenen Audit: ISO 27001 Beratung mit festem Scope für Startups und Mittelstand. Realistische Zeitpläne statt Papier-ISMS.",
		}
		tmpl.ExecuteTemplate(w, "iso27001-de.html", data)
	})

	http.HandleFunc("/de/nis2", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "NIS2 Beratung: Betroffenheit & Umsetzung | GovaGuard",
			Description: "NIS2 betrifft mehr Unternehmen als gedacht: Betroffenheit in 60 Sekunden prüfen, Pflichten verstehen, Umsetzung priorisieren — pragmatisch statt Papier-Compliance.",
		}
		tmpl.ExecuteTemplate(w, "nis2.html", data)
	})

	// NIS2 Betroffenheits-Check (htmx endpoint; simplified first assessment, not legal advice)
	http.HandleFunc("/de/nis2/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		if err := r.ParseForm(); err != nil {
			w.Write([]byte(errorFragment("Eingaben konnten nicht verarbeitet werden. Bitte versuchen Sie es erneut.")))
			return
		}

		sector := r.FormValue("sector")
		employees := r.FormValue("employees")
		revenue := r.FormValue("revenue")
		special := r.FormValue("special")

		annex1 := strings.HasPrefix(sector, "a1:")
		annex2 := strings.HasPrefix(sector, "a2:")
		large := employees == "gte250" || revenue == "gt50"
		medium := !large && (employees == "50-249" || revenue == "10-50")

		const disclaimer = `<p class="check-disclaimer">Unverbindliche Ersteinschätzung auf Basis Ihrer Angaben — keine Rechtsberatung. Die genaue Einstufung hängt von Tätigkeiten, Registrierungspflichten und dem finalen Stand des NIS2UmsuCG ab.</p>`
		const cta = `<p><a href="/?need=nis2#contact" class="framework-link">Sprechen Sie mit uns über die nächsten Schritte →</a></p>`

		var result string
		switch {
		case special == "always" || special == "kritis":
			result = `<div class="check-result-box check-in">
				<h3>Voraussichtlich in den Anwendungsbereich — unabhängig von der Unternehmensgröße</h3>
				<p>Für DNS-Dienste, TLD-Registries, Vertrauensdiensteanbieter, TK-Anbieter und KRITIS-Betreiber gelten die NIS2-Pflichten größenunabhängig — in der Regel als besonders wichtige Einrichtung: Registrierung, Risikomanagement nach Art. 21, Meldepflichten (24 h / 72 h / 1 Monat) und persönliche Pflichten der Geschäftsleitung.</p>` + cta + disclaimer + `</div>`
		case !annex1 && !annex2:
			result = `<div class="check-result-box check-out">
				<h3>Voraussichtlich nicht direkt betroffen</h3>
				<p>Ihr Sektor fällt nach Ihren Angaben nicht unter Anlage I oder II. Beachten Sie aber die Lieferkette: NIS2-pflichtige Kunden müssen die Sicherheit ihrer Zulieferer vertraglich absichern (Art. 21 Abs. 2 lit. d) — Nachweise wie ISO 27001 werden dadurch faktisch auch für nicht regulierte Anbieter zur Vertriebsvoraussetzung.</p>` + cta + disclaimer + `</div>`
		case !large && !medium:
			result = `<div class="check-result-box check-out">
				<h3>Voraussichtlich unterhalb der Größenschwellen</h3>
				<p>Mit weniger als 50 Mitarbeitenden und unter 10 Mio. € Umsatz sind Sie in der Regel nicht direkt erfasst — Ausnahmen gelten für besondere Diensteanbieter (DNS, TLD, Vertrauensdienste, TK). Über Kundenverträge in der Lieferkette können NIS2-Anforderungen Sie dennoch erreichen.</p>` + cta + disclaimer + `</div>`
		case annex1 && large:
			result = `<div class="check-result-box check-in">
				<h3>Voraussichtlich besonders wichtige Einrichtung</h3>
				<p>Anlage-I-Sektor und Großunternehmen: Es gilt der volle Pflichtenkatalog — Registrierung beim BSI, Risikomanagement nach Art. 21, Meldepflichten (24 h / 72 h / 1 Monat), Billigungs- und Überwachungspflicht der Geschäftsleitung. Bußgeldrahmen bis 10 Mio. € oder 2 % des weltweiten Umsatzes.</p>` + cta + disclaimer + `</div>`
		default:
			result = `<div class="check-result-box check-in">
				<h3>Voraussichtlich wichtige Einrichtung</h3>
				<p>Ihr Sektor fällt unter NIS2 und Ihre Unternehmensgröße liegt über den Schwellenwerten: Der Maßnahmenkatalog nach Art. 21 und die Meldepflichten gelten auch für Sie — mit abgestufter Aufsicht und einem Bußgeldrahmen bis 7 Mio. € oder 1,4 % des weltweiten Umsatzes.</p>` + cta + disclaimer + `</div>`
		}
		w.Write([]byte(result))
	})

	// Imprint page
	http.HandleFunc("/imprint", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "Imprint - GovaGuard",
			Description: "Legal information and imprint for GovaGuard, CTO & CISO security consulting from Cologne, Germany.",
		}
		tmpl.ExecuteTemplate(w, "imprint.html", data)
	})

	// Privacy Policy page
	http.HandleFunc("/privacy", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "Privacy Policy - GovaGuard",
			Description: "How GovaGuard handles personal data: GDPR-compliant privacy policy for govaguard.com.",
		}
		tmpl.ExecuteTemplate(w, "privacy.html", data)
	})

	// SEO plumbing
	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://govaguard.com/</loc></url>
  <url><loc>https://govaguard.com/iso-27001</loc></url>
  <url><loc>https://govaguard.com/soc-2</loc></url>
  <url><loc>https://govaguard.com/compare/soc-2-vs-iso-27001</loc></url>
  <url><loc>https://govaguard.com/de/c5-testat</loc></url>
  <url><loc>https://govaguard.com/de/iso-27001-beratung</loc></url>
  <url><loc>https://govaguard.com/de/nis2</loc></url>
  <url><loc>https://govaguard.com/dora</loc></url>
  <url><loc>https://govaguard.com/hipaa</loc></url>
  <url><loc>https://govaguard.com/iso-42001</loc></url>
  <url><loc>https://govaguard.com/cra</loc></url>
  <url><loc>https://govaguard.com/de/c5-vs-iso-27001</loc></url>
  <url><loc>https://govaguard.com/imprint</loc></url>
  <url><loc>https://govaguard.com/privacy</loc></url>
</urlset>
`))
	})

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("User-agent: *\nAllow: /\n\nSitemap: https://govaguard.com/sitemap.xml\n"))
	})

	// HTMX endpoint for contact form
	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html")

		if err := r.ParseForm(); err != nil {
			w.Write([]byte(errorFragment("Failed to process form. Please try again.")))
			return
		}

		// Honeypot: hidden field humans never fill in. Pretend success so bots
		// can't tell they were filtered.
		if r.FormValue("website") != "" {
			w.Write([]byte(successFragment))
			return
		}

		// Extract form data
		form := ContactForm{
			Name:       strings.TrimSpace(r.FormValue("name")),
			Email:      strings.TrimSpace(r.FormValue("email")),
			Company:    strings.TrimSpace(r.FormValue("company")),
			Message:    strings.TrimSpace(r.FormValue("message")),
			Framework:  strings.TrimSpace(r.FormValue("framework")),
			SourcePage: strings.TrimSpace(r.FormValue("source_page")),
		}
		if len(form.SourcePage) > 200 {
			form.SourcePage = form.SourcePage[:200]
		}

		// Validate required fields
		if form.Name == "" || form.Email == "" || form.Message == "" {
			w.Write([]byte(errorFragment("Please fill in all required fields.")))
			return
		}

		if len(form.Name) > 200 || len(form.Company) > 200 || len(form.Message) > 5000 {
			w.Write([]byte(errorFragment("Your message is too long. Please shorten it and try again.")))
			return
		}

		// Require a bare address: reject name-addr forms like "Jane <jane@x.com>"
		// so the CRM's Email property only ever holds a usable address.
		addr, err := mail.ParseAddress(form.Email)
		if err != nil || len(form.Email) > 254 || addr.Address != form.Email {
			w.Write([]byte(errorFragment("Please enter a valid email address.")))
			return
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(notionPage(notionDatabaseID, form)); err != nil {
			log.Print(err)
			w.Write([]byte(errorFragment("Failed to send message. Please try again later.")))
			return
		}

		// Detach from the request context: a visitor closing the tab mid-submit
		// must not cancel the CRM write. The client's 10s timeout still bounds it.
		request, err := http.NewRequestWithContext(context.WithoutCancel(r.Context()), http.MethodPost, notionAPIURL, &buf)
		if err != nil {
			log.Print(err)
			w.Write([]byte(errorFragment("Failed to send message. Please try again later.")))
			return
		}

		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+notionAPIKey)
		request.Header.Set("Notion-Version", notionVersion)

		response, err := client.Do(request)
		if err != nil {
			log.Print(err)
			w.Write([]byte(errorFragment("Failed to send message. Please try again later or email us directly.")))
			return
		}
		defer response.Body.Close()

		if response.StatusCode < 200 || response.StatusCode > 299 {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
			log.Printf("notion API returned %d: %s", response.StatusCode, body)
			w.Write([]byte(errorFragment("Failed to send message. Please try again later or email us directly.")))
			return
		}

		// Success response
		w.Write([]byte(successFragment))
	})

	log.Println("GovaGuard landing page running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
