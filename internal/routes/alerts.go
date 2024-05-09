package routes

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal"
	"github.com/vladwithcode/juzgados/internal/alerts"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

func RegisterAlertRoutes(router *httprouter.Router) {
	router.GET("/alerta/:id", auth.WithAuthMiddleware(RenderSingleAlertPage))

	router.GET("/api/alerts/all", TestAllAlerts)
	router.POST("/api/alerts", auth.WithAuthMiddleware(CreateAlert))

	router.GET("/api/alerts/report/:userId", GetReportForUser)
	router.POST("/api/alerts/report/:userId", CreatePDFForReport)
	router.PUT("/api/alerts", auth.WithAuthMiddleware(UpdateAlertsForUser))
	router.DELETE("/api/alert/:id", auth.WithAuthMiddleware(DeleteAlertById))

	// Update the accord data for the alert with the provided id
	router.PUT("/api/alert-refresh/:id", auth.WithAuthMiddleware(RefreshAlertById))
}

func CreateAlert(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	err := r.ParseForm()

	if err != nil {
		fmt.Printf("[Parse Form Err]: %v\n", err)
		respondWithError(w, 400, "La información proporcionada no es válida")
		return
	}

	var (
		caseId     string = r.Form.Get("caseId")
		natureCode string = r.Form.Get("natureCode")
		userId     string = auth.Id
	)

	doc, _ := tsj.GetCaseData(caseId, natureCode, nil, tsj.DEFAULT_DAYS_BACK)
	alert := db.Alert{
		UserId:        userId,
		CaseId:        db.TrimField(caseId),
		NatureCode:    db.TrimField(natureCode),
		LastCheckedAt: time.Now(),
		LastUpdatedAt: time.Now(),
		Active:        true,
	}

	if doc != nil {
		alert.LastAccord.String = doc.Accord
		alert.LastAccord.Valid = true
		alert.LastAccordDate.Time = doc.AccordDate
		alert.LastAccordDate.Valid = true
		alert.Nature = doc.Nature
	}

	_, err = db.CreateAlertWithData(&alert)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			respondWithError(w, 400, "Ya existe una alerta para este caso en tu perfil")
			return
		}
		fmt.Printf("[Create Err]: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error al crear la alerta")
		return
	}

	templ, err := template.New("alert-card.html").Funcs(template.FuncMap{
		"FormatDate": internal.FormatDate,
	}).ParseFiles("web/templates/alert-card.html")

	if err != nil {
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	err = templ.ExecuteTemplate(w, "alert-card", alert)

	if err != nil {
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}
}

func RenderSingleAlertPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, auth *auth.Auth) {
	id := ps.ByName("id")
	user, err := db.GetUserById(auth.Id)

	if err != nil {
		fmt.Printf("[Find user err]: %v\n", err)
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	alert, err := db.FindAlertById(id)

	if err != nil {
		fmt.Printf("[Find alert err]: %v\n", err)
		w.Header().Add("Content-Type", "text/html")

		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(404)
			msg := fmt.Sprintf("<p>No se encontró alerta con id: %v</p>", id)
			w.Write([]byte(msg))
		}
		w.WriteHeader(500)
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	if alert.UserId != user.Id {
		fmt.Printf("Alert ID: %v;   User ID: %v\n", alert.UserId, user.Id)
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(403)
		w.Write([]byte("<p>La alerta solicitada no pertenece al usuario actual</p>"))
		return
	}

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
		"FormatDate": internal.FormatDate,
		"GetNature": func(code string) string {
			return internal.CodesMap[code]
		},
	}).ParseFiles("web/templates/layout.html", "web/templates/alerts/single-alert.html")

	if err != nil {
		fmt.Printf("[Parse err]: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	data := map[string]any{
		"User":  user,
		"Alert": alert,
	}

	err = templ.Execute(w, data)

	if err != nil {
		fmt.Printf("[Execute err]: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}
}

func RefreshAlertById(w http.ResponseWriter, r *http.Request, ps httprouter.Params, auth *auth.Auth) {
	id := ps.ByName("id")
	alert, err := db.FindAlertById(id)

	// For htmx request to
	w.Header().Add("HX-Reswap", "beforeend")

	if err != nil {
		fmt.Printf("[Find err]: %v\n", err)
		w.WriteHeader(404)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>No se encontró la alerta especificada</p>"))
		return
	}

	templ, err := template.New("blocks.html").ParseFiles("web/templates/blocks.html")

	if err != nil {
		fmt.Printf("[Parse template err]: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	doc, err := tsj.GetCaseData(alert.CaseId, alert.NatureCode, nil, tsj.DEFAULT_DAYS_BACK)

	if err != nil {
		var NotFoundErr *tsj.NotFoundError
		if errors.As(err, &NotFoundErr) {
			w.WriteHeader(404)
			w.Header().Add("Content-Type", "text/html")

			err = templ.ExecuteTemplate(w, "error-card", map[string]any{
				"Message":   "No se encontró nueva información para este caso en los ultimos 30 días",
				"BtnLabel":  "Aceptar",
				"ErrorCode": 404,
			})

			if err == nil {
				return
			}
		}

		fmt.Printf("GetCaseData err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	alert.Nature = doc.Nature
	alert.LastAccord.String = doc.Accord
	alert.LastAccord.Valid = doc.Accord != ""
	alert.LastAccordDate.Time = doc.AccordDate
	alert.LastAccordDate.Valid = doc.AccordDate != (time.Time{})

	err = db.UpdateAlertAccord(auth.Id, alert.CaseId, alert.NatureCode, alert)

	if err != nil {
		fmt.Printf("UpdateAlert err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")

		err = templ.ExecuteTemplate(w, "error-card", map[string]any{
			"Message":   "No se pudo actualizar la alerta",
			"BtnLabel":  "Aceptar",
			"ErrorCode": 500,
		})

		if err != nil {
			fmt.Printf("Execute update err: %v\n", err)
			w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		}
		return
	}

	templ, err = template.New("single-alert.html").Funcs(template.FuncMap{
		"GetNature": func(code string) string {
			return internal.CodesMap[code]
		},
		"FormatDate": internal.FormatDate,
	}).ParseFiles("web/templates/alerts/single-alert.html")

	if err != nil {
		fmt.Printf("Parse Single err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("Ocurrió un error inesperado"))
		return
	}

	w.Header().Set("HX-Reswap", "innerHTML")
	err = templ.ExecuteTemplate(w, "alert-data", map[string]any{
		"Alert": alert,
	})

	if err != nil {
		fmt.Printf("Execute alert-data err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("Ocurrió un error inesperado"))
	}
}

func UpdateAlertsForUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	alerts, err := db.FindAlertsByUser(auth.Id, true)
	if err != nil {
		fmt.Printf("[Find err]: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	caseKeys := []string{}
	alertMap := make(map[string]*db.Alert)
	for _, alert := range alerts {
		cK := alert.CaseId + "+" + alert.NatureCode
		caseKeys = append(caseKeys, cK)
		alertMap[cK] = alert
	}

	docs, err := tsj.GetCasesData(caseKeys, tsj.DEFAULT_DAYS_BACK, time.Now())

	if err != nil {
		fmt.Printf("[GetCasesData err]: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	for _, doc := range docs.Docs {
		cK := doc.Case + "+" + doc.NatureCode
		alert := alertMap[cK]
		alert.LastAccord.String = doc.Accord
		alert.LastAccord.Valid = true
		alert.LastAccordDate.Time = doc.AccordDate
		alert.LastAccordDate.Valid = true
		alert.Nature = doc.Nature
	}

	err = db.UpdateAlertAccords(alerts)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	templ, err := template.New("alert-card.html").Funcs(template.FuncMap{
		"FormatDate": internal.FormatDate,
	}).ParseFiles("web/templates/alert-card.html")

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	err = templ.ExecuteTemplate(w, "alert-cards", alerts)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}
}

func DeleteAlertById(w http.ResponseWriter, r *http.Request, ps httprouter.Params, auth *auth.Auth) {
	id := ps.ByName("id")
	err := db.DeleteUserAlertById(id, auth.Id)

	if err != nil {
		fmt.Printf("[Find alert err]: %v\n", err)
		w.Header().Add("Content-Type", "text/html")

		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(404)
			msg := fmt.Sprintf("<p>No se encontró alerta con id: %v</p>", id)
			w.Write([]byte(msg))
		}

		w.WriteHeader(500)
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	templ, err := template.New("blocks.html").ParseFiles("web/templates/blocks.html")

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}

	data := map[string]any{
		"Message":     "Se eliminó con exito la alerta especificada",
		"ErrorCode":   0,
		"ButtonLabel": "",
	}
	err = templ.ExecuteTemplate(w, "success-card", data)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
		return
	}
}

// subscriberMeta is a struct holding a pointer to the subscriber User
// aswell as the position in User.Alerts of the alert it will update
type subscriberMeta struct {
	alertPos int
	userPtr  *db.AutoReportUser
}

// subscriberMap is a caseKey (ie caseId-natureCode) to []subscriberMeta map
type subscriberMap map[string][]subscriberMeta

func TestAllAlerts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userAlerts, err := db.FindAutoReportAlertsWithUserData()

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 500, "Ocurrio un error en el servidor")
		return
	}

	subscribers := subscriberMap{}
	caseKeys := []string{}

	for _, user := range userAlerts {
		for alertIdx, alert := range user.Alerts {
			cK := fmt.Sprintf("%v-%v", alert.CaseId, alert.NatureCode)

			suscriber := subscriberMeta{
				alertPos: alertIdx,
				userPtr:  user,
			}

			if _, ok := subscribers[cK]; ok {
				subscribers[cK] = append(subscribers[cK], suscriber)
			} else {
				subscribers[cK] = []subscriberMeta{suscriber}
			}
		}
	}

	for k := range subscribers {
		caseKeys = append(caseKeys, k)
	}

	resCases, err := tsj.GetCasesData(caseKeys, 30, time.Now())

	if err != nil {
		fmt.Printf("GetCases err: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error en el servidor")
		return
	}

	for _, c := range resCases.Docs {
		caseId := strings.TrimSpace(strings.TrimLeft(c.Case, "0"))
		cK := fmt.Sprintf("%v-%v", caseId, c.NatureCode)

		if subs, ok := subscribers[cK]; ok {
			for _, sub := range subs {
				al := &sub.userPtr.Alerts[sub.alertPos]
				al.LastAccord.String = c.Accord
				al.LastAccord.Valid = true
				al.LastAccordDate.Time = c.AccordDate
				al.LastAccordDate.Valid = true
			}
		}
	}

	wg := sync.WaitGroup{}

	for _, user := range userAlerts {
		wg.Add(1)
		go func(user *db.AutoReportUser) {
			defer wg.Done()
			docPath, err := alerts.GenReportPdfWithData(*user)

			if err != nil {
				fmt.Printf("GenReport err: %v\n", err)
				return
			}

			pdfUrl := fmt.Sprintf("%v://%v%v", r.URL.Scheme, r.URL.Hostname(), docPath)
			fmt.Printf("pdfUrl: %v\n", pdfUrl)

			// err = whatsapp.SendReportMessage(*user, "http://tsjdgo.gob.mx/Recursos/images/flash/ListasAcuerdos/1132024/fam2.pdf")

			if err != nil {
				fmt.Printf("GenReport err: %v\n", err)
				return
			}
		}(user)
	}

	wg.Wait()

	respondWithJSON(w, 200, map[string]any{"users": userAlerts})
}
