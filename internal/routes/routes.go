package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/reader"
)

func NewRouter() http.Handler {
	router := httprouter.New()

	router.GET("/", indexHandler)
	router.GET("/api/file", getFile)
	router.GET("/api/case", searchCase)

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

func searchCase(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	date := r.URL.Query().Get("date")
	caseType := r.URL.Query().Get("type")

	dateSegments := strings.Split(date, "-")
	dateAsInt, err := strconv.Atoi(dateSegments[2])

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 400, "La fecha es inválida")
		return
	}

	date = strconv.Itoa(dateAsInt) + dateSegments[1] + dateSegments[0]

	pdfContent, err := reader.Reader(date, caseType)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	caseId := r.URL.Query().Get("id")
	searchExp, err := reader.GenRegExp(caseId)

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Se introdujo un caso inválido")
		return
	}

	idx := searchExp.FindIndex(*pdfContent)

	if len(idx) == 0 {
		respondWithError(w, 500, "No se encontró información sobre el caso solicitado")
		return
	}

	start, end := idx[0], idx[1]

	type successResponse struct {
		Index   string `json:"index"`
		Content string `json:"content"`
	}

	contentAsStr := []byte(*pdfContent)[start:end]

	lineExp := regexp.MustCompile("(?m)\n")
	spaceExp := regexp.MustCompile(" {2,}")
	rows := lineExp.Split(string(contentAsStr), -1)
	var data = map[string]string{
		"idx":    "",
		"case":   "",
		"nature": "",
		"accord": "",
	}

	for i, str := range rows {
		cols := spaceExp.Split(str, -1)

		if i == 0 {
			data["idx"] = cols[0]
			data["case"] = cols[1]
			data["nature"] = cols[2]
			data["accord"] = cols[3]
			continue
		}

		if len(cols) > 2 {
			data["nature"] = data["nature"] + "\n" + cols[1]
			data["accord"] = data["accord"] + "\n" + cols[2]
			continue
		}

		data["accord"] = data["accord"] + "\n" + cols[1]
	}

	rowTempl, err := template.ParseFiles("web/templates/table-row.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't parse row")
	}

	rowTempl.Execute(w, data)
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
