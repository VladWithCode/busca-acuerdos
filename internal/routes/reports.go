package routes

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal"
	"github.com/vladwithcode/juzgados/internal/alerts"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
	"github.com/vladwithcode/juzgados/internal/whatsapp"
)

func GetReportForUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userId := ps.ByName("userId")
	alerts, err := FindAlerts(userId)

	if err != nil {
		fmt.Printf("FindAlerts: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error en el servidor")
		return
	}

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
		"FormatDate": func(date time.Time) string {
			var (
				d    int    = date.Day()
				m    int    = int(date.Month())
				y    int    = date.Year()
				mStr string = fmt.Sprint(m)
				dStr string = fmt.Sprint(d)
			)

			if m < 10 {
				mStr = fmt.Sprintf("0%d", m)
			}

			if d < 10 {
				dStr = fmt.Sprintf("0%d", d)
			}

			return fmt.Sprintf("%v-%v-%v", dStr, mStr, y)
		},
		"GetNature": func(nc string) string {
			return internal.CodesMap[nc]
		},
	}).ParseFiles("web/templates/reports/layout.html", "web/templates/reports/alert-report.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
		return
	}

	err = templ.Execute(w, map[string]any{
		"Alerts": *alerts,
	})

	if err != nil {
		fmt.Printf("Execute err: %v\n", err)
	}
}

func ReportHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params, auth *auth.Auth) {
	alerts, err := FindAlerts(auth.Id)

	if err != nil {
		fmt.Printf("FindAlerts: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error en el servidor")
		return
	}

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
		"FormatDate": func(date time.Time) string {
			var (
				d    int    = date.Day()
				m    int    = int(date.Month())
				y    int    = date.Year()
				mStr string = fmt.Sprint(m)
				dStr string = fmt.Sprint(d)
			)

			if m < 10 {
				mStr = fmt.Sprintf("0%d", m)
			}

			if d < 10 {
				dStr = fmt.Sprintf("0%d", d)
			}

			return fmt.Sprintf("%v-%v-%v", dStr, mStr, y)
		},
		"GetNature": func(nc string) string {
			return internal.CodesMap[nc]
		},
	}).ParseFiles("web/templates/reports/layout.html", "web/templates/reports/alert-report.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
	}

	err = templ.Execute(w, map[string]any{
		"Alerts": *alerts,
	})

	if err != nil {
		fmt.Printf("Execute err: %v\n", err)
	}
}

func CreatePDFForReport(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userId := ps.ByName("userId")
	docPath, err := alerts.GenReportPdf(userId)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 500, fmt.Sprintf("Error al crear reporte para usuario con id: %v", userId))
		return
	}

	respondWithJSON(w, 201, fmt.Sprintf("Documento disponible en %v%v%v", r.URL.Scheme, r.URL.Hostname(), docPath))
}

func SendTestMessage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		headerVars []whatsapp.TemplateVar
		bodyVars   []whatsapp.TemplateVar
	)

	y, m, d := time.Now().Date()

	var (
		dStr string
		mStr string
	)

	if d < 10 {
		dStr = fmt.Sprintf("0%d", d)
	} else {
		dStr = fmt.Sprintf("%d", d)
	}

	if m < 10 {
		mStr = fmt.Sprintf("0%d", m)
	} else {
		mStr = fmt.Sprintf("%d", m)
	}

	headerVars = append(headerVars, whatsapp.TemplateVar{
		"type": "document",
		"document": struct {
			Link     string `json:"link"`
			Filename string `json:"filename"`
		}{Link: "https://www.postgresql.org/files/documentation/pdf/16/postgresql-16-US.pdf", Filename: "reporte-.pdf"},
	})

	bodyVars = append(bodyVars, whatsapp.TemplateVar{
		"type": "text",
		"text": "Jairo Rangel",
	})
	bodyVars = append(bodyVars, whatsapp.TemplateVar{
		"type": "date_time",
		"date_time": struct {
			FallbackValue string `json:"fallback_value"`
		}{
			FallbackValue: fmt.Sprintf("%v-%v-%v", y, mStr, dStr),
		},
	})

	err := whatsapp.SendTemplateMessage("+526183188452", whatsapp.TemplateData{
		TemplateName: "report_file",
		HeaderVars:   headerVars,
		BodyVars:     bodyVars,
	})

	w.Header().Add("Content-Type", "text/html")

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("<p>Error</p>"))

		return
	}

	w.WriteHeader(200)
	w.Write([]byte("<p>Success</p>"))
}

func FindAlerts(userId string) (*[]db.Alert, error) {
	alerts, err := db.FindAutoReportAlertsForUser(userId)

	if err != nil {
		return nil, err
	}

	var caseKeys []string
	foundAlertMap := map[string]*db.Alert{}
	for _, alert := range *alerts {
		cK := alert.CaseId + "-" + alert.NatureCode

		caseKeys = append(caseKeys, cK)
		foundAlertMap[cK] = &alert
	}
	fmt.Printf("Fetching for %d cases\n", len(caseKeys))
	result, err := tsj.GetCasesData(caseKeys, tsj.DEFAULT_DAYS_BACK)

	if err != nil {
		return nil, err
	}

	resAlerts := []db.Alert{}

	for _, doc := range result.Docs {
		if doc == nil {
			continue
		}

		// Remove white-space & leading 0s
		cK := strings.TrimLeft(strings.TrimSpace(doc.Case), "0") + "-" + doc.NatureCode

		foundAlertMap[cK].LastAccord.Valid = true
		foundAlertMap[cK].LastAccord.String = doc.Accord
		foundAlertMap[cK].LastAccordDate.Valid = true
		foundAlertMap[cK].LastAccordDate.Time = doc.AccordDate
		foundAlertMap[cK].LastUpdatedAt = time.Now()
		foundAlertMap[cK].LastCheckedAt = time.Now()

		resAlerts = append(resAlerts, *foundAlertMap[cK])
	}

	return &resAlerts, nil
}
