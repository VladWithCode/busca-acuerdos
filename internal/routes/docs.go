package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/db"
)

func RegisterDocRoutes(router *httprouter.Router) {
	router.GET("/api/docs", getDocs)
	router.GET("/api/docs/by-case/:caseID", getDocByCase)
	router.POST("/api/doc", createDoc)
	router.GET("/api/doc/:ID", getDocByID)
}

func getDocByCase(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rawCaseId := ps.ByName("caseID")

	caseID, err := url.PathUnescape(rawCaseId)

	if err != nil {
		respondWithError(w, 500, "El expediente ingresado es invalido")
		return
	}

	doc, err := db.GetDocByCase(caseID)

	if err != nil {
		respondWithError(w, 500, "No se encontró entrada para el caso solicitado")
		return
	}

	respondWithJSON(w, 200, doc)
}

func getDocByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("ID")
	doc, err := db.GetDocByID(id)

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "No se encontró el documento solicitado")
		return
	}

	respondWithJSON(w, 200, doc)
}

func getDocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	docs, err := db.GetDocs()

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "No se pudo recuperar los documentos")
		return
	}

	respondWithJSON(w, 200, docs)
}

func createDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data := db.Doc{}
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&data)

	if err != nil {
		fmt.Printf("Malformed json data: %v\n", err)
		respondWithError(w, 400, "La información proporcionada es inválida")
		return
	}

	err = db.CreateDoc(data.ID, data.Case, data.Nature, data.NatureCode, data.Accord, data.AccordDate)

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "No se pudo crear el documento")
		return
	}

	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte("<p>Creación exitosa</p>"))
}
