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

func searchCase(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	caseID := r.URL.Query().Get("id")
	caseType := r.URL.Query().Get("type")

	d := time.Now()

	doc, err := tsj.GetCaseData(caseID, caseType, &d, tsj.DEFAULT_DAYS_BACK)

	rowTempl, err := template.New("case-card.html").Funcs(template.FuncMap{
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
	}).ParseFiles("web/templates/case-card.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't parse row")
		return
	}

	rowTempl.Execute(w, []db.Doc{*doc})
}

func searchCases(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cases := r.URL.Query()["cases"]
	result, err := tsj.GetCasesData(cases, tsj.DEFAULT_DAYS_BACK)

	if len(result.NotFoundKeys) == len(cases) {
		respondWithError(w, 500, "No se encontró ningun documento solicitado")
		return
	}

	templ, err := template.New("case-card.html").Funcs(template.FuncMap{
		"FormatDate": func(date time.Time) string {
			return fmt.Sprintf("%d-%s-%d", date.Day(), date.Month(), date.Year())
		},
	}).ParseFiles("web/templates/case-card.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't parse row")

		return
	}

	templ.Execute(w, result.Docs)
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
