package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultNotionAPIURL = "https://api.notion.com/v1/pages"
	notionVersion       = "2022-06-28"
	// Google Calendar appointment page; override with BOOKING_URL.
	defaultBookingURL = "https://calendar.app.google/pwQCqiCQhpNpkCXT7"
)

//go:embed templates/* static/*
var content embed.FS

type PageData struct {
	Title       string
	Description string
	// BookingURL enables the "book an intro call" CTAs; empty hides them
	// and the primary CTA falls back to the contact form.
	BookingURL string
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

// insightPage drives the /insights article routes and the sitemap.
type insightPage struct {
	path, tmplName, title, desc string
}

var insightPages = []insightPage{
	{"/insights", "insights.html", "Insights — Security Certification Knowledge | GovaGuard", "Working knowledge from our certification practice: what regulations require, what audits cost, and what holds up when the auditor arrives. Sources cited."},
	{"/de/insights/393-sgb-v-erklaert", "insight-393.html", "§ 393 SGB V erklärt: C5-Pflicht im Gesundheitswesen | GovaGuard", "Wer von § 393 SGB V betroffen ist, was ab wann gilt und wie Cloud-Anbieter im Gesundheitswesen zum C5-Testat kommen — mit Primärquellen."},
	{"/de/insights/c5-testat-kosten", "insight-c5-kosten.html", "C5-Testat Kosten: Der ehrliche Überblick | GovaGuard", "Was ein C5-Testat wirklich kostet: die zwei Kostenblöcke, die echten Kostentreiber und die legitimen Hebel, den Aufwand zu senken."},
	{"/de/insights/nis2umsucg-stand", "insight-nis2umsucg.html", "NIS2UmsuCG: Aktueller Stand & Pflichten | GovaGuard", "Wo das deutsche NIS2-Umsetzungsgesetz heute steht: Fristen, Registrierung, Pflichten — laufend aktualisiert, mit Primärquellen."},
	{"/de/insights/nis2-artikel-21-checkliste", "insight-nis2-art21.html", "NIS2 Artikel 21: Maßnahmen-Checkliste | GovaGuard", "Die zehn Maßnahmenbereiche aus Art. 21 Abs. 2 NIS2 als Checkliste — und welcher Teil eines ISO-27001-ISMS sie jeweils abdeckt."},
	{"/de/insights/nis2-haftung-geschaeftsleitung", "insight-nis2-haftung.html", "NIS2-Haftung: Pflichten der Geschäftsleitung | GovaGuard", "Umsetzungs-, Überwachungs- und Schulungspflichten der Geschäftsleitung nach § 38 BSIG — und was persönliche Haftung konkret bedeutet."},
	{"/de/insights/nis2-lieferkette", "insight-nis2-lieferkette.html", "NIS2 in der Lieferkette: Pflichten für Zulieferer | GovaGuard", "Wie NIS2-Anforderungen über Kundenverträge auch nicht regulierte Zulieferer erreichen — und welche Nachweise Einkäufer jetzt verlangen."},
	{"/de/insights/c5-kriterienkatalog", "insight-c5-katalog.html", "C5-Kriterienkatalog im Überblick | GovaGuard", "Der BSI C5-Kriterienkatalog in Klartext: Aufbau, Basis- und Zusatzkriterien, Umgebungsparameter — und was Prüfer daraus machen."},
	{"/de/insights/c5-typ1-zu-typ2", "insight-c5-typ2.html", "C5 Typ 1 zu Typ 2: Der Weg in der Praxis | GovaGuard", "Vom Stichtags-Testat zur Wirksamkeitsprüfung: Prüfungszeitraum, Nachweisführung und die Fristen aus § 393 SGB V für das Gesundheitswesen."},
	{"/insights/paper-isms-warning-signs", "insight-paper-isms.html", "The Paper ISMS: Seven Warning Signs | GovaGuard", "Seven signs your ISO 27001 program exists only on paper — and why Stage 2 auditors, surveillance audits, and customer reviews take it apart."},
	{"/insights/grc-tools-what-they-dont-do", "insight-grc-tools.html", "GRC Platforms: What They Don't Do | GovaGuard", "Vanta, Drata & Co. automate evidence brilliantly. What they don't do: your risk assessment, your SoA, your management review. An honest map."},
	{"/insights/iso-27001-soa-guide", "insight-soa.html", "ISO 27001 Statement of Applicability Guide | GovaGuard", "The SoA is the document auditors and customers actually read: structure, example entries, and the mistakes that cause audit findings."},
	{"/insights/choosing-certification-body", "insight-cert-body.html", "Choosing an ISO 27001 Certification Body | GovaGuard", "Accreditation, pricing, and red flags: how to pick a certification body whose certificate your customers will accept — and cert mills to avoid."},
	{"/insights/soc-2-evidence-list", "insight-soc2-evidence.html", "SOC 2 Evidence List: What Auditors Request | GovaGuard", "The evidence categories behind a SOC 2 Type II: what auditors ask for, which artifacts satisfy them, and what to automate versus collect by hand."},
	{"/insights/cpa-firm-from-europe", "insight-cpa-europe.html", "Choosing a SOC 2 Auditor from Europe | GovaGuard", "Any licensed CPA firm can attest a European company. Selection criteria, remote fieldwork reality, pricing, and the red flags to walk away from."},
	{"/insights/soc-2-observation-window", "insight-soc2-window.html", "SOC 2 Observation Window: 3 or 12 Months? | GovaGuard", "The Type II observation window decision: what buyers accept, what a short first window costs later, and the pragmatic path most SaaS teams take."},
	{"/insights/baa-guide-eu-vendors", "insight-baa.html", "BAA Guide for EU Health-Tech Vendors | GovaGuard", "Business associate agreements for European vendors entering US healthcare: the chain of liability, subcontractors, and clauses to negotiate."},
	{"/insights/dora-article-30-clauses", "insight-dora30.html", "DORA Article 30 Clauses for Vendors | GovaGuard", "The contract clauses financial customers must impose on ICT vendors under DORA Article 30 — and how to sign them without overpromising."},
	{"/insights/hitrust-e1-i1-r2", "insight-hitrust.html", "HITRUST e1, i1 or r2: Which Tier? | GovaGuard", "The three HITRUST assessment tiers compared: control counts, effort, what hospital procurement actually asks for, and where to start."},
	{"/insights/eucs-status", "insight-eucs.html", "EUCS Status: What It Means for C5 | GovaGuard", "The EU cloud certification scheme is still not adopted. Current status, the sovereignty debate, and why BSI C5 remains the German benchmark."},
	{"/insights/cra-ce-marking-checklist", "insight-cra-checklist.html", "CRA CE-Marking Checklist to December 2027 | GovaGuard", "Fifteen months to CE marking for software: the quarter-by-quarter checklist, the conformity-route decision, and the reporting duty already live."},
	{"/insights/cra-sbom-in-ci", "insight-cra-sbom.html", "SBOM in CI for CRA Compliance | GovaGuard", "Meeting the CRA's SBOM duty in practice: CycloneDX vs SPDX, wiring generation into CI, and linking findings to the 24-hour ENISA clock."},
	{"/insights/cra-open-source-steward", "insight-cra-oss.html", "Open Source & the CRA: Steward or Manufacturer | GovaGuard", "When open-source involvement becomes commercial activity under the CRA: contributor, steward, or manufacturer — and the duties each role carries."},
	{"/insights/iso-42001-vs-eu-ai-act", "insight-42001-aiact.html", "ISO 42001 vs EU AI Act: The Difference | GovaGuard", "One is law, one is a certificate. What ISO 42001 evidences under the EU AI Act, what it cannot replace, and the pragmatic sequence for vendors."},
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

// submitToNotion writes one contact-form submission into the Notion database.
// Detached from the request context: a visitor closing the tab mid-submit must
// not cancel the CRM write; the client's 10s timeout still bounds it.
func submitToNotion(ctx context.Context, client *http.Client, apiURL, apiKey, databaseID string, form ContactForm) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(notionPage(databaseID, form)); err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(context.WithoutCancel(ctx), http.MethodPost, apiURL, &buf)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Notion-Version", notionVersion)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		log.Printf("notion API returned %d: %s", response.StatusCode, body)
		return errors.New("notion API error")
	}
	return nil
}

func errorFragment(msg string) string {
	return `
		<div class="error-message">
			<div class="error-icon">✗</div>
			<p>` + template.HTMLEscapeString(msg) + `</p>
		</div>
	`
}

// --- Cost & timeline estimator ------------------------------------------------
//
// Every number below is a range published on this site (ISO 27001 page, C5 cost
// article, NIS2 pages) or a cited market range. The two profile answers (team
// size, infrastructure) only pick the lower half, full range, or upper half of
// a published range — they never invent figures. NIS2 carries no audit fee by
// design: it is law, not a certificate; its effort reuses the published ISO
// bands via the site's "Art. 21 is structurally covered by ISO 27001" claim.

func buildEstimatorResult(fw string, team int, infra, bookingURL string) string {
	var name string
	switch fw {
	case "iso-27001":
		name = "ISO 27001"
	case "c5":
		name = "BSI C5"
	case "nis2":
		name = "NIS2"
	default:
		return errorFragment("Please pick a framework first.")
	}

	// Profile score: team size and infrastructure each nudge by one; only a
	// clear signal in both (|score| >= 2) narrows a published range to a half.
	score := 0
	switch {
	case team < 20:
		score--
	case team > 200:
		score++
	}
	switch infra {
	case "cloud":
		score--
	case "onprem":
		score++
	}
	sub := func(lo, hi int) (int, int) {
		mid := (lo + hi) / 2
		switch {
		case score <= -2:
			return lo, mid
		case score >= 2:
			return mid, hi
		}
		return lo, hi
	}
	rng := func(lo, hi int) string { return strconv.Itoa(lo) + "–" + strconv.Itoa(hi) }
	kEUR := func(lo, hi int) string { return "€" + strconv.Itoa(lo) + "k–" + strconv.Itoa(hi) + "k" }
	fte := func(lo, hi int) string {
		return "0." + strconv.Itoa(lo) + "–0." + strconv.Itoa(hi)
	}

	teamLabel := strconv.Itoa(team)
	if team >= 300 {
		teamLabel = "300+"
	}

	var b strings.Builder
	row := func(icon, iconClass, text string) {
		b.WriteString(`<div class="terminal-line output"><span class="` + iconClass + `">` + icon + `</span><span class="est-row">` + text + `</span></div>`)
	}
	stat := func(number, label string) {
		b.WriteString(`<div class="stat"><div class="stat-number">` + number + `</div><div class="stat-label">` + label + `</div></div>`)
	}

	b.WriteString(`<div class="terminal est-result"><div class="terminal-header"><span class="terminal-dot"></span><span class="terminal-dot"></span><span class="terminal-dot"></span><span class="terminal-title">estimate_` + fw + `.txt</span></div><div class="terminal-body">`)
	b.WriteString(`<div class="terminal-line"><span class="prompt">$</span><span class="command">./estimate --team ` + teamLabel + ` --infra ` + infra + ` --framework ` + fw + `</span></div>`)

	// Cost chart rows: dim segment = published range, bright overlay = the band
	// the profile lands in (bandLo == bandHi == 0 → range-only, nothing to narrow).
	type chartRow struct {
		label                  string
		lo, hi, bandLo, bandHi int
	}
	var chartRows []chartRow
	var axisMax int

	b.WriteString(`<div class="est-stats">`)
	switch fw {
	case "iso-27001":
		mLo, mHi := sub(4, 8)
		fLo, fHi := sub(5, 15)
		eLo, eHi := sub(2, 5)
		stat(rng(mLo, mHi), "months to certificate (published 4–8)")
		stat(kEUR(fLo, fHi), "certification body (published €5k–15k)")
		stat(fte(eLo, eHi), "FTE internal during the build (published 0.2–0.5)")
		chartRows = []chartRow{{"certification body", 5, 15, fLo, fHi}}
		axisMax = 15
	case "c5":
		mLo, mHi := sub(3, 6)
		tLo, tHi := sub(40, 120)
		stat(rng(mLo, mHi), "months to Typ-1 readiness (published 3–6)")
		stat(kEUR(tLo, tHi), "Testat fee (published €40k–120k, Typ 1 at the lower end)")
		stat("6–12", "months audit period for Typ 2 (published)")
		chartRows = []chartRow{
			{"gap analysis (market)", 5, 25, 0, 0},
			{"implementation (market)", 20, 60, 0, 0},
			{"Testat fee", 40, 120, tLo, tHi},
		}
		axisMax = 120
	case "nis2":
		mLo, mHi := sub(4, 8)
		eLo, eHi := sub(2, 5)
		stat(rng(mLo, mHi), "months to implement Art. 21 (via the published ISO 27001 band)")
		stat(fte(eLo, eHi), "FTE internal during the build (via the published ISO 27001 band)")
		stat("€0", "mandatory audit fee — NIS2 is law, not a certificate")
		chartRows = []chartRow{{"ISO 27001 certificate (optional evidence)", 5, 15, 0, 0}}
		axisMax = 15
	}
	b.WriteString(`</div>`)
	b.WriteString(`<p class="est-disclaim">Estimates, not quotes — your answers only shift results within published ranges. Sources under "show our math".</p>`)

	// Cost picture: one shared axis per framework so bar lengths stay comparable.
	pct := func(v int) string { return strconv.FormatFloat(float64(v)*100/float64(axisMax), 'f', 1, 64) }
	b.WriteString(`<div class="est-chart"><p class="est-heading">// COST PICTURE</p>`)
	for _, r := range chartRows {
		b.WriteString(`<div class="est-chart-row"><span class="est-chart-label">` + r.label + `</span><div class="est-chart-track">`)
		b.WriteString(`<span class="est-chart-range" style="left:` + pct(r.lo) + `%;width:` + pct(r.hi-r.lo) + `%" title="published ` + kEUR(r.lo, r.hi) + `"></span>`)
		value := kEUR(r.lo, r.hi)
		if r.bandLo != 0 || r.bandHi != 0 {
			b.WriteString(`<span class="est-chart-band" style="left:` + pct(r.bandLo) + `%;width:` + pct(r.bandHi-r.bandLo) + `%" title="your profile ` + kEUR(r.bandLo, r.bandHi) + `"></span>`)
			value = kEUR(r.bandLo, r.bandHi)
		}
		b.WriteString(`</div><span class="est-chart-value">` + value + `</span></div>`)
	}
	b.WriteString(`<p class="est-chart-key"><span class="est-key-swatch est-key-range"></span>published range<span class="est-key-swatch est-key-band"></span>your profile</p></div>`)

	// Profile drivers
	switch {
	case team < 20:
		row("✓", "success", "under 20 people: expect the lower ends of these ranges")
	case team <= 80:
		row("→", "info", "20–80 people: typically the lower half of these ranges")
	case team <= 200:
		row("→", "info", "80–200 people: plan for the upper half of these ranges")
	default:
		row("!", "warning", "over 200 people: outside our standard case (20–200) — talk to us first")
	}
	switch infra {
	case "cloud":
		row("✓", "success", "cloud-only: standardized evidence pulls the effort down")
	case "onprem":
		row("!", "warning", "on-premise estate: more infrastructure in scope")
	default:
		row("→", "info", "hybrid estate: mid-range profile")
	}
	switch fw {
	case "iso-27001":
		row("→", "info", "consulting: fixed scope, quoted after the gap analysis — no fantasy pricing")
	case "c5":
		row("→", "info", "readiness on top: gap analysis €5k–25k + implementation €20k–60k (market, ingenieur.de 2025)")
		row("✓", "success", "existing ISO 27001? a lived ISMS shortens the path to Typ-1 readiness")
	case "nis2":
		switch {
		case team < 50:
			row("→", "info", "under 50 people: usually not directly in scope — customer contracts can still pass NIS2 duties through the supply chain")
		case team < 250:
			row("!", "warning", "50–249 people: likely an important entity if your sector is in Annex I/II — fines up to €7M or 1.4% of worldwide turnover")
		default:
			row("!", "warning", "250+ people: likely essential or important entity depending on sector — fines up to €10M or 2% of worldwide turnover")
		}
		row("→", "info", `your sector decides — run the 60-second check: <a href="/de/nis2">NIS2 Betroffenheits-Check (DE)</a>`)
		row("!", "warning", "NIS2UmsuCG is in force since 6 Dec 2025; the BSI registration window (3 months, § 33 BSIG) has already closed for in-scope entities")
	}

	b.WriteString(`<p class="est-heading">// PROJECT PHASES</p><ul class="est-phases">`)
	switch fw {
	case "iso-27001":
		b.WriteString(`<li>gap analysis — 2–3 weeks</li><li>risk assessment &amp; ISMS build — 2–4 months</li><li>internal audit &amp; management review</li><li>certification audit — Stage 1 + 2</li>`)
	case "c5":
		b.WriteString(`<li>gap analysis against the C5 catalog</li><li>implementation &amp; evidence setup</li><li>Typ 1 Testat — point in time</li><li>6–12 month audit period → Typ 2 Testat</li>`)
	case "nis2":
		b.WriteString(`<li>scope &amp; registration check (§ 33 BSIG)</li><li>gap analysis against the ten Art. 21 measure areas</li><li>ISMS build — most of Art. 21 is structurally covered by ISO 27001</li><li>incident reporting drill — 24 h / 72 h / 1 month</li>`)
	}
	b.WriteString(`</ul>`)

	b.WriteString(`<div class="est-ctas">`)
	if bookingURL != "" {
		b.WriteString(`<a href="` + bookingURL + `" target="_blank" rel="noopener" class="btn btn-primary" data-umami-event="book-call"><span class="btn-text">Book an intro call</span><span class="btn-arrow">→</span></a>`)
	}
	b.WriteString(`<a href="/?need=` + fw + `#contact" class="est-form-link">Or use the secure form →</a></div>`)

	b.WriteString(`<details class="est-math"><summary>Show our math — sources and labels</summary><ul><li>Your two answers only shift results within published ranges — lower half, full range, or upper half. They never invent new figures.</li>`)
	switch fw {
	case "iso-27001":
		b.WriteString(`<li>Timeline (4–8 months), certification-body fees (€5k–15k) and the 0.2–0.5 FTE band are practice estimates, not a quote: <a href="/iso-27001">ISO 27001 — timeline &amp; cost expectations</a></li>`)
	case "c5":
		b.WriteString(`<li>Gap analysis (€5k–25k), implementation (€20k–60k) and the €40k–120k Testat fee are market ranges (ingenieur.de, 2025); 3–6 months to Typ-1 readiness and the 6–12 month audit period are practice figures: <a href="/de/insights/c5-testat-kosten">C5-Testat Kosten (DE)</a> · <a href="/de/c5-testat">C5-Testat Beratung (DE)</a></li>`)
	case "nis2":
		b.WriteString(`<li>NIS2 mandates no certification audit; effort reuses the published ISO 27001 bands (4–8 months, 0.2–0.5 FTE) because a lived ISO 27001 ISMS structurally covers most of Art. 21: <a href="/de/insights/nis2-artikel-21-checkliste">Art. 21 Checkliste (DE)</a> · <a href="/iso-27001">ISO 27001</a></li><li>Entity classes, fines and deadlines: <a href="/de/insights/nis2umsucg-stand">NIS2UmsuCG Stand (DE)</a></li>`)
	}
	b.WriteString(`<li>Your scope decides where you land in these ranges; the gap analysis turns the ` + name + ` estimate into a fixed quote.</li></ul></details>`)

	b.WriteString(`</div></div>`)
	return b.String()
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

	bookingURL := os.Getenv("BOOKING_URL")
	if bookingURL == "" {
		bookingURL = defaultBookingURL
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
			Description: "Founder-led consulting for security certifications: ISO 27001, SOC 2 and BSI C5, from gap analysis to passed audit. Fixed scope, direct founder access.",
			BookingURL:  bookingURL,
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

		const disclaimer = `<p class="check-disclaimer">Unverbindliche Ersteinschätzung auf Basis Ihrer Angaben — keine Rechtsberatung. Die genaue Einstufung hängt von Ihren konkreten Tätigkeiten ab. Das NIS2UmsuCG ist seit dem 6. Dezember 2025 in Kraft; die Registrierungsfrist beim BSI ist bereits abgelaufen.</p>`
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
				<p>Anlage-I-Sektor und Großunternehmen: Es gilt der volle Pflichtenkatalog — Registrierung beim BSI, Risikomanagement nach Art. 21, Meldepflichten (24 h / 72 h / 1 Monat), Umsetzungs- und Überwachungspflicht der Geschäftsleitung (§ 38 BSIG). Bußgeldrahmen bis 10 Mio. € oder 2 % des weltweiten Umsatzes.</p>` + cta + disclaimer + `</div>`
		default:
			result = `<div class="check-result-box check-in">
				<h3>Voraussichtlich wichtige Einrichtung</h3>
				<p>Ihr Sektor fällt unter NIS2 und Ihre Unternehmensgröße liegt über den Schwellenwerten: Der Maßnahmenkatalog nach Art. 21 und die Meldepflichten gelten auch für Sie — mit abgestufter Aufsicht und einem Bußgeldrahmen bis 7 Mio. € oder 1,4 % des weltweiten Umsatzes.</p>` + cta + disclaimer + `</div>`
		}
		w.Write([]byte(result))
	})

	// Cost & timeline estimator (see buildEstimatorResult)
	http.HandleFunc("/cost-estimator", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "Cost & Timeline Estimator: ISO 27001, BSI C5 & NIS2 | GovaGuard",
			Description: "Two questions, honest ranges: estimate cost and timeline for ISO 27001, BSI C5 or NIS2 — from published figures and cited market data. No email required.",
			BookingURL:  bookingURL,
		}
		tmpl.ExecuteTemplate(w, "estimator.html", data)
	})

	// htmx endpoint: computes the estimate server-side
	http.HandleFunc("/cost-estimator/estimate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		if err := r.ParseForm(); err != nil {
			w.Write([]byte(errorFragment("Could not process your input. Please try again.")))
			return
		}
		team, err := strconv.Atoi(r.FormValue("team"))
		if err != nil || team < 1 {
			team = 80
		}
		if team > 300 {
			team = 300
		}
		infra := r.FormValue("infrastructure")
		if infra != "cloud" && infra != "onprem" {
			infra = "hybrid"
		}
		w.Write([]byte(buildEstimatorResult(r.FormValue("framework"), team, infra, bookingURL)))
	})

	// Insights hub + articles (data-driven; see insightPages)
	for _, p := range insightPages {
		http.HandleFunc(p.path, func(w http.ResponseWriter, r *http.Request) {
			tmpl.ExecuteTemplate(w, p.tmplName, PageData{Title: p.title, Description: p.desc})
		})
	}

	// Imprint page
	http.HandleFunc("/imprint", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:       "Imprint - GovaGuard",
			Description: "Legal information and imprint for GovaGuard, security certification consulting from Cologne, Germany.",
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
		staticPaths := []string{
			"/", "/iso-27001", "/soc-2", "/dora", "/hipaa", "/iso-42001", "/cra",
			"/compare/soc-2-vs-iso-27001", "/de/c5-testat", "/de/iso-27001-beratung",
			"/de/nis2", "/de/c5-vs-iso-27001", "/cost-estimator",
			"/imprint", "/privacy",
		}
		var b strings.Builder
		b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
		for _, u := range staticPaths {
			b.WriteString("  <url><loc>https://govaguard.com" + u + "</loc></url>\n")
		}
		for _, p := range insightPages {
			b.WriteString("  <url><loc>https://govaguard.com" + p.path + "</loc></url>\n")
		}
		b.WriteString("</urlset>\n")
		w.Write([]byte(b.String()))
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

		if err := submitToNotion(r.Context(), client, notionAPIURL, notionAPIKey, notionDatabaseID, form); err != nil {
			log.Print(err)
			w.Write([]byte(errorFragment("Failed to send message. Please try again later or email us directly.")))
			return
		}

		// Success response
		w.Write([]byte(successFragment))
	})

	log.Println("GovaGuard landing page running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
