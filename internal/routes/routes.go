package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/reader"
	"github.com/vladwithcode/juzgados/internal/tsj"
)

var Router *httprouter.Router

func NewRouter() http.Handler {
	router := httprouter.New()

	// Static Routes
	router.GET("/", indexHandler)
	router.GET("/dashboard", auth.WithAuthMiddleware(dashboardHandler))

	// API Routes
	router.GET("/api/docs", getDocs)
	router.POST("/api/docs", createDoc)
	router.GET("/api/file", getFile)
	router.GET("/api/case", searchCase)
	router.GET("/api/cases", searchCases)
	router.GET("/api/docs-by-case/:caseID", getDocByCase)
	router.GET("/api/docs/:ID", getDocByID)

	// User Routes
	router.GET("/iniciar-sesion", SignInHandler)
	router.POST("/sign-in", SignInUser)
	router.POST("/api/users", CreateUser)

	// Report Routes
	router.GET("/report", auth.WithAuthMiddleware(ReportHandler))

	// Alert Routes
	router.GET("/api/alerts/all", TestAllAlerts)
	router.POST("/api/alerts", auth.WithAuthMiddleware(CreateAlert))
	//router.POST("/api/alerts/test", SendTestMessage)
	router.GET("/api/alerts/report/:userId", GetReportForUser)
	router.POST("/api/alerts/report/:userId", CreatePDFForReport)

	router.GET("/api/cases/accord", auth.WithAuthMiddleware(SearchAccord))

	router.NotFound = http.FileServer(http.Dir("web/static"))

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/index.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
	}

	templ.Execute(w, nil)
}

func getFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	caseType := r.URL.Query().Get("type")
	date := r.URL.Query().Get("date")

	if date == "" || caseType == "" {
		respondWithError(w, 400, "La fecha y caso son requeridas")
		return
	}

	segments := strings.Split(date, "-")

	day, _ := strconv.Atoi(segments[2])
	month, _ := strconv.Atoi(segments[1])

	date = fmt.Sprintf("%d%d%s", day, month, segments[0])
	fmt.Printf("date: %v\n", date)

	content, err := reader.Reader(date, caseType)

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't read file")
		return
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

func findCaseInPast(startDate time.Time, caseID, caseType string, responseCh chan<- *db.Doc, dateCh chan<- *time.Time) {

	// Set Date to previous day
	startDate = startDate.Local().AddDate(0, 0, -1)

	resultDoc := &db.Doc{}

	for i := 0; i < 31; i++ {
		year, month, date := startDate.Date()
		searchDate := fmt.Sprintf("%d%d%d", date, month, year)

		contentAsStr, err := tsj.FetchAndReadDoc(caseID, searchDate, caseType)

		if err != nil {
			if i == 30 {
				break
			}

			startDate = startDate.Local().AddDate(0, 0, -1)

			continue
		}

		resultDoc = tsj.DataToDoc(contentAsStr)
		break
	}

	if empDoc := (db.Doc{}); *resultDoc == empDoc {
		responseCh <- &empDoc
		return
	}

	dateCh <- &startDate
	resultDoc.AccordDate = startDate
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

		doc.AccordDate = startDate
		doc.NatureCode = caseType
	} else if empDoc != *dbDoc {
		doc = dbDoc
		doc.NatureCode = caseType
	} else {
		dateCh := make(chan *time.Time)
		go findCaseInPast(startDate, caseID, caseType, fetchDocCh, dateCh)

		fetchDoc = <-fetchDocCh
		accordDate := <-dateCh

		if *fetchDoc == (empDoc) {
			respondWithError(w, 404, "No se encontró información del expediente solicitado en el ultimo mes")
			return
		}

		doc = fetchDoc

		doc.AccordDate = *accordDate
		doc.NatureCode = caseType
	}

	rowTempl, err := template.New("table-row.html").Funcs(template.FuncMap{
		"FormatDate": func(date time.Time) string {
			var (
				d    int    = date.Day()
				m    int    = int(date.Month())
				y    int    = date.Year()
				mStr string = fmt.Sprint(m)
				dStr string = fmt.Sprint(d)
			)

			if m < 10 {
				mStr = fmt.Sprintf("0%d", m)
			}

			if d < 10 {
				dStr = fmt.Sprintf("0%d", d)
			}

			return fmt.Sprintf("%v-%v-%v", dStr, mStr, y)
		},
		"Trim": func(str string) string {
			return strings.TrimSpace(str)
		},
	}).ParseFiles("web/templates/table-row.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't parse row")
		return
	}

	rowTempl.Execute(w, *doc)
}

func searchCases(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cases := r.URL.Query()["cases"]
	resultsCh := make(chan *db.Doc, len(cases))
	var resultDocs []db.Doc

	// Since findCaseInPast starts the day before use tomorrows date
	// TODO: change findCaseInPast to use the date supplied
	startDate := time.Now().Local().AddDate(0, 0, 1)

	for _, c := range cases {
		fakeCh := make(chan *time.Time, 5)

		params := strings.Split(c, "+")

		go func() {
			findCaseInPast(startDate, params[0], params[1], resultsCh, fakeCh)
		}()
	}

	for res := range resultsCh {
		resultDocs = append(resultDocs, *res)

		if len(resultDocs) == len(cases) {
			break
		}
	}

	emptyCount := 0
	for _, doc := range resultDocs {
		if doc == (db.Doc{}) {
			emptyCount++
		}
	}

	if emptyCount == len(cases) {
		respondWithError(w, 500, "No se encontró ningun documento solicitado")
		return
	}

	templ, err := template.New("table-rows.html").Funcs(template.FuncMap{
		"FormatDate": func(date time.Time) string {
			return fmt.Sprintf("%d-%d-%d", date.Day(), date.Month(), date.Year())
		},
		"Trim": func(str string) string {
			return strings.TrimSpace(str)
		},
	}).ParseFiles("web/templates/table-rows.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't parse row")

		return
	}

	templ.Execute(w, resultDocs)
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
