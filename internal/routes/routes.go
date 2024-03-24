package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/reader"
)

func NewRouter() http.Handler {
	router := httprouter.New()

	// Static Routes
	router.GET("/", auth.CheckAuthMiddleware(indexHandler))
	router.GET("/error/500", auth.CheckAuthMiddleware(Render500Error))

	router.GET("/tests", tests)

	// API Routes
	router.GET("/api/file", getFile)

	// Doc Routes
	RegisterDocRoutes(router)
	// Case Routes
	RegisterCaseRoutes(router)
	// User Routes
	RegisterUserRoutes(router)
	// Report Routes
	RegisterReportRoutes(router)
	// Alert Routes
	RegisterAlertRoutes(router)

	// Serve static content
	router.NotFound = http.FileServer(http.Dir("web/static"))

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/index.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
func Render500Error(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/500.html")

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		w.Write([]byte("Ha ocurrido un error inesperado en el servidor"))
		return
	}

	data := map[string]any{
		"User": auth,
	}

	templ.Execute(w, data)
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

	content, err := reader.Reader(date, caseType)

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't read file")
		return
	}

	fmt.Fprintln(w, string(*content))
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
