package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

func SearchAccord(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	err := r.ParseForm()

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 400, "La peticion contiene información inválida")
		return
	}

	searchParams := r.Form.Get("searchParams")
	idx, err := strconv.Atoi(r.Form.Get("idx"))

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 400, "La peticion contiene información inválida")
		return
	}

	params := strings.Split(searchParams, "-")
	caseId := params[0]
	natureCode := params[1]

	// Start search in TSJ
	doc, err := tsj.GetCaseData(caseId, natureCode, nil, 31)

	if err != nil {
		fmt.Printf("GetCase Err: %v\n", err)

		if strings.Contains(err.Error(), "No se encontró") {
			respondWithError(w, 404, err.Error())
			return
		}

		respondWithError(w, 500, "Ocurrio un error en el servidor")
		return
	}

	alert := db.Alert{
		NatureCode:     natureCode,
		LastAccord:     sql.NullString{Valid: true, String: doc.Accord},
		LastAccordDate: sql.NullTime{Time: doc.AccordDate, Valid: true},
		CaseId:         doc.Case,
		LastUpdatedAt:  time.Now(),
		LastCheckedAt:  time.Now(),
	}
	evenRow := idx%2 == 0

	data := map[string]any{}

	data["Alert"] = alert
	data["I"] = idx
	data["EvenRow"] = evenRow

	tmpl, err := template.New("dashboard.html").Funcs(template.FuncMap{
		"IsEven": func(n int) bool {
			return n%2 == 0
		},
		"GetNature": func(nc string) string {
			return internal.CodesMap[nc]
		},
		// Refer to https://stackoverflow.com/questions/18276173/calling-a-template-with-several-pipeline-parameters
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}).ParseFiles("web/templates/dashboard.html")

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error en el servidor")
		return
	}

	err = tmpl.ExecuteTemplate(w, "accord", data)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 500, "Ocurrió un error en el servidor")
		return
	}

	// Save alert data to DB
	err = db.UpdateAlertAccord(auth.Id, caseId, natureCode, &alert)

	if err != nil {
		fmt.Printf("Update Alert Err: %v\n", err)
	}
}
