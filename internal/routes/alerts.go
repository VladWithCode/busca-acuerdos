package routes

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/alerts"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

func RegisterAlertRoutes(router *httprouter.Router) {
	router.GET("/api/alerts/all", TestAllAlerts)
	router.POST("/api/alerts", auth.WithAuthMiddleware(CreateAlert))

	router.GET("/api/alerts/report/:userId", GetReportForUser)
	router.POST("/api/alerts/report/:userId", CreatePDFForReport)
}

func CreateAlert(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	err := r.ParseForm()

	if err != nil {
		fmt.Printf("[Parse Form Err]: %v\n", err)
		respondWithError(w, 400, "La información proporcionada no es válida")
		return
	}

	fmt.Printf("r.Form: %+v\n", r.Form)

	var (
		caseId     string = r.Form.Get("caseId")
		natureCode string = r.Form.Get("natureCode")
		userId     string = auth.Id
	)

	alert, err := db.CreateAlert(userId, caseId, natureCode)

	if err != nil {
		fmt.Printf("[Create Err]: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error al crear la alerta")
		return
	}

	respondWithJSON(w, 201, alert)
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

	resCases, err := tsj.GetCasesData(caseKeys, 30)

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
