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

	dbPool, err := db.Connect()

	if err != nil {
		log.Printf("Error while connecting to DB: %v", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	log.Println("Query active alerts")
	alerts, err := db.FindDistinctActiveAlerts()

	if err != nil {
		log.Printf("Find alerts err: %v\n", err)
		os.Exit(1)
	}

	caseKeys := []string{}
	keyMap := make(map[string]bool)

	for _, al := range alerts {
		cK := al.GetCaseKey()

		if !keyMap[cK] {
			keyMap[cK] = true
			caseKeys = append(caseKeys, cK)
		}
	}

	log.Println("Fetch cases data")
	resCases, err := tsj.GetCasesData(caseKeys, 0)

	if err != nil {
		log.Printf("GetCases err: %v\n", err)
		os.Exit(1)
	}

	log.Println("Update db alerts")
	err = db.UpdateAlertsForCases(resCases.Docs)

	if err != nil {
		log.Printf("Update alerts err: %v\n", err)
		os.Exit(1)
	}

	log.Println("Updated Alerts successfully")
	os.Exit(0)
}
