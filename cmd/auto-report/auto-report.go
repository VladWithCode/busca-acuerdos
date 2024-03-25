package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/vladwithcode/juzgados/internal/alerts"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
	"github.com/vladwithcode/juzgados/internal/whatsapp"
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
	homePath, err := os.UserHomeDir()

	if err != nil {
		log.Println("Error: Couldn't load UserHomeDir")
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Open Log err: %v\n", err)
		os.Exit(1)
	}

	stdOut, err := os.OpenFile(homePath+"/.local/log/auto-report.log/log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Open Log err: %v\n", err)
		os.Exit(1)
	}

	stdErr, err := os.OpenFile(homePath+"/.local/log/auto-report.log/error", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Open Err err: %v\n", err)
		os.Exit(1)
	}

	if stdOut != nil && stdErr != nil {
		os.Stdout = stdOut
		os.Stderr = stdErr
	}

	tsjDir := os.Getenv("TSJ_DIR")

	err = godotenv.Load(fmt.Sprintf("%v/.env", tsjDir))


	if err != nil {
		log.Printf("Error: Couldn't load enviroment %v\n", err)
		os.Exit(1)
	}
	hostname := os.Getenv("TSJ_SITE_HOSTNAME")

	if hostname == "" {
		log.Println("Error: env hostname is missing")
		os.Exit(1)
	}

	dbPool, err := db.Connect()

	if err != nil {
		log.Printf("Error while connecting to DB: %v", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	userAlerts, err := db.FindAutoReportAlertsWithUserData()

	if err != nil {
		log.Printf("Find alerts err: %v\n", err)
		os.Exit(1)
	}

	subscribers := subscriberMap{}
	caseKeys := []string{}

	for _, user := range userAlerts {
		for alertIdx, alert := range user.Alerts {
			cK := alert.GetCaseKey()

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
		log.Printf("GetCases err: %v\n", err)
		os.Exit(1)
	}

	for _, c := range resCases.Docs {
		caseId := strings.TrimSpace(strings.TrimLeft(c.Case, "0"))
		cK := fmt.Sprintf("%v+%v", caseId, c.NatureCode)

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
			_, err := alerts.GenReportPdfWithData(*user)

			if err != nil {
				log.Printf("GenReport err: %v\n", err)
				return
			}

			docHref := fmt.Sprintf("http://%v/reports/%v/report.pdf", hostname, user.Id)
			err = whatsapp.SendReportMessage(*user, docHref)

			if err != nil {
				log.Printf("SendReport err: %v\n", err)
				return
			}
		}(user)
	}

	wg.Wait()

	os.Exit(0)
}
