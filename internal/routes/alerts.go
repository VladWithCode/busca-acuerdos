package routes

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
)

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

func TestAllAlerts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	alerts, err := db.FindAllAutoReportAlerts()

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 500, "Ocurrio un error en el servidor")
		return
	}

	respondWithJSON(w, 200, map[string]any{"alerts": *alerts})
}
