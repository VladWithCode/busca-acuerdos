package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

func main() {
	log.Println("Start alert auto-update")
	homePath, err := os.UserHomeDir()

	if err != nil {
		log.Println("Error: Couldn't load UserHomeDir")
		os.Exit(1)
	}

	stdErr, err := os.OpenFile(homePath+"/.local/log/tsj/auto-update.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Open log err: %v\n", err)
		os.Exit(1)
	}

	if stdErr != nil {
		os.Stderr = stdErr
	}

	tsjDir := os.Getenv("TSJ_DIR")

	err = godotenv.Load(fmt.Sprintf("%v/.env", tsjDir))

	if err != nil {
		log.Printf("Error: Couldn't load enviroment %v\n", err)
		os.Exit(1)
	}

	dbPool, err := db.Connect()

	if err != nil {
		log.Printf("Error while connecting to DB: %v", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	log.Println("Querying active alerts")
	alerts, err := db.FindDistinctActiveAlerts()
	log.Printf("Found %v active alerts", len(alerts))

	if err != nil {
		log.Printf("Find alerts err: %v\n", err)
		os.Exit(1)
	}

	caseKeys := []string{}
	keyMap := make(map[string]bool)

	for _, al := range alerts {
		cK := al.GetCaseKey()

		if seen, ok := keyMap[cK]; !ok || !seen {
			keyMap[cK] = true
			caseKeys = append(caseKeys, cK)
		}
	}

	log.Println("Fetching cases data")
	resCases, err := tsj.GetCasesData(caseKeys, 15)
	log.Printf("Found data for %v cases\n", len(resCases.Docs))

	if err != nil {
		log.Printf("GetCases err: %v\n", err)
		os.Exit(1)
	}

	log.Println("Updating db alerts")
	err, updatedCount, errs := db.UpdateAlertsForCases(resCases.Docs)

	log.Printf("Updated %v alerts successfully\n", updatedCount)

	if len(errs) > 0 {
		log.Printf("%v errors occurred while updating db", len(errs))
		log.Printf("Last error: %v", errs[len(errs)-1])
	}

	if err != nil {
		log.Printf("Update alerts err: %v\n", err)
		os.Exit(1)
	}

	log.Println("Updated Alerts successfully")
	os.Exit(0)
}
