package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/vladwithcode/juzgados/internal/alerts"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

// subscriberMeta is a struct holding a pointer to the subscriber User
// aswell as the position in User.Alerts of the alert it will update
type subscriberMeta struct {
	alertPos int
	userPtr  *db.AutoReportUser
}

// subscriberMap is a caseKey (ie caseId-natureCode) to []subscriberMeta map
type subscriberMap map[string][]subscriberMeta

func main() {
	godotenv.Load(".env")
	dbPool, err := db.Connect()

	if err != nil {
		fmt.Printf("Error while connecting to DB: %v", err)
		return
	}
	defer dbPool.Close()

	userAlerts, err := db.FindAutoReportAlertsWithUserData()

	if err != nil {
		fmt.Printf("err: %v\n", err)
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

			fmt.Printf("docPath: %v\n", docPath)
			if err != nil {
				fmt.Printf("GenReport err: %v\n", err)
				return
			}

			//pdfUrl := fmt.Sprintf("%v://%v%v", r.URL.Scheme, r.URL.Hostname(), docPath)

			// err = whatsapp.SendReportMessage(*user, "http://tsjdgo.gob.mx/Recursos/images/flash/ListasAcuerdos/1132024/fam2.pdf")

			// if err != nil {
			// 	fmt.Printf("GenReport err: %v\n", err)
			// 	return
			// }
		}(user)
	}

	wg.Wait()
}
