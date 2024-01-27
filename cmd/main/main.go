package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/routes"
)

func main() {
	godotenv.Load(".env")
	router := routes.NewRouter()

	dbPool, err := db.Connect()
	if err != nil {
		fmt.Printf("Error while connecting to DB: %v", err)
	}
	defer dbPool.Close()

	portString := os.Getenv("PORT")

	if portString == "" {
		log.Fatal("Port is not set in env")
	}

	addr := fmt.Sprintf(":%s", portString)
	fmt.Printf("Server listening on http://localhost%s\n", addr)

	err = http.ListenAndServe(addr, router)

	if err != nil {
		log.Fatal(err)
	}
}
