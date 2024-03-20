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
)

func RegisterReportRoutes(router *httprouter.Router) {
	router.GET("/report", auth.WithAuthMiddleware(ReportHandler))
}

func GetReportForUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userId := ps.ByName("userId")
	alerts, err := FindAlerts(userId)

	if err != nil {
		fmt.Printf("FindAlerts: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error en el servidor")
		return
	}

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
		"FormatDate": internal.FormatDate,
		"GetNature": func(nc string) string {
			return internal.CodesMap[nc]
		},
	}).ParseFiles("web/templates/reports/layout.html", "web/templates/reports/alert-report.html", "web/templates/reports/css.html")

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
		"FormatDate": internal.FormatDate,
		"GetNature": func(nc string) string {
			return internal.CodesMap[nc]
		},
	}).ParseFiles("web/templates/reports/layout.html", "web/templates/reports/alert-report.html", "web/templates/reports/css.html")

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
