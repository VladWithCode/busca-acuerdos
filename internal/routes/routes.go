package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	db "github.com/vladwithcode/juzgados/internal/db/docs"
	"github.com/vladwithcode/juzgados/internal/reader"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

func NewRouter() http.Handler {
	router := httprouter.New()

	router.GET("/", indexHandler)
	router.GET("/api/docs", getDocs)
	router.POST("/api/docs", createDoc)
	router.GET("/api/file", getFile)
	router.GET("/api/case", searchCase)
	router.GET("/api/docs-by-case/:caseID", getDocByCase)
	router.GET("/api/docs/:ID", getDocByID)

	router.NotFound = http.FileServer(http.Dir("web/static"))

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	templ, err := template.ParseFiles("web/templates/layout.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
	}

	templ.Execute(w, nil)
}

func getFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	content, err := reader.Reader("4122023", "civ2")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't read file")
	}

	fmt.Fprintln(w, string(*content))
}

func waitForGetDoc(wg *sync.WaitGroup, caseID string, responseCh chan<- *db.Doc) {
	wg.Add(1)
	defer wg.Done()

	doc, err := db.GetDocByCase(caseID)

	if err != nil {
		fmt.Printf("[GetDoc err]: %v\n", err)
		responseCh <- &db.Doc{}
		return
	}

	responseCh <- doc
	return
}

func waitForFetchDoc(wg *sync.WaitGroup, caseID, searchDate, caseType string, responseCh chan<- *db.Doc) {
	wg.Add(1)
	defer wg.Done()

	contentAsStr, err := tsj.FetchAndReadDoc(caseID, searchDate, caseType)

	if err != nil {
		fmt.Printf("[FetchDoc err]: %v\n", err)
		responseCh <- &db.Doc{}
		return
	}

	doc := tsj.DataToDoc(contentAsStr)

	responseCh <- doc
}

func findCaseInPast(startDate time.Time, caseID, caseType string, responseCh chan<- *db.Doc) {
	// Set Date to previous day
	startDate = startDate.Local().AddDate(0, 0, -1)

	resultDoc := &db.Doc{}

	for i := 0; i < 14; i++ {
		year, month, date := startDate.Date()
		searchDate := fmt.Sprintf("%d%d%d", date, month, year)

		contentAsStr, err := tsj.FetchAndReadDoc(caseID, searchDate, caseType)

		if err != nil {
			fmt.Printf("[FindInPast err]: %v\n", err)

			startDate = startDate.Local().AddDate(0, 0, -1)

			continue
		}

		resultDoc = tsj.DataToDoc(contentAsStr)
		// Here it sends the Doc resulting from DataToDoc
		responseCh <- resultDoc
		return
	}

	// Here it sends the empty Doc
	responseCh <- resultDoc
}

func searchCase(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	caseID := r.URL.Query().Get("id")
	caseType := r.URL.Query().Get("type")

	wg := sync.WaitGroup{}

	dbDocCh := make(chan *db.Doc)
	fetchDocCh := make(chan *db.Doc)

	startDate := time.Now()
	var year, month, date = startDate.Date()

	searchDate := fmt.Sprintf("%d%d%d", date, month, year)

	go waitForGetDoc(&wg, caseID, dbDocCh)
	go waitForFetchDoc(&wg, caseID, searchDate, caseType, fetchDocCh)

	wg.Wait()

	rowTempl, err := template.ParseFiles("web/templates/table-row.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't parse row")
	}

	// Read & close dbDocCh
	dbDoc := <-dbDocCh
	close(dbDocCh)
	// fetchDoc stays open for fetching past docs
	fetchDoc := <-fetchDocCh

	var doc *db.Doc

	if empDoc := (db.Doc{}); (empDoc) != *fetchDoc {
		doc = fetchDoc
		// Since we alredy got the doc we can close fetchDocCh
		close(fetchDocCh)
	} else if empDoc != *dbDoc {
		doc = dbDoc
	} else {
		go findCaseInPast(startDate, caseID, caseType, fetchDocCh)

		fetchDoc = <-fetchDocCh

		if *fetchDoc == (db.Doc{}) {
			respondWithError(w, 404, "No se encontró información del expediente solicitado en los ultimos 14 días")
			return
		}

		doc = fetchDoc
	}

	rowTempl.Execute(w, *doc)
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

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)

	if err != nil {
		log.Printf("Failed to marshal JSON response %v", payload)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Println("Responding with 5xx error: ", msg)
	}

	type errResponse struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, errResponse{
		Error: msg,
	})
}
