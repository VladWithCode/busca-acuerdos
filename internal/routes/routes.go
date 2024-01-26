package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/reader"
)

func NewRouter() http.Handler {
	router := httprouter.New()

	router.GET("/", indexHandler)
	router.GET("/api/data/:name", apiDataHandler)
	router.GET("/api/file", getFile)
	router.GET("/api/case", searchCase)

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// fmt.Fprintln(w, "Welcome to homepage!")
	templ, err := template.ParseFiles("internal/templates/layout.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
	}

	templ.Execute(w, nil)
}

func apiDataHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "This string appends the `name` param: Your name is %s", ps.ByName("name"))
}

func getFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	content, err := reader.Reader()

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't read file")
	}

	fmt.Fprintln(w, string(*content))
}

func searchCase(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pdfContent, err := reader.Reader()

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Couldn't read pdf")
	}

	caseId := r.URL.Query().Get("id")
	searchExp, err := reader.GenRegExp(caseId)

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Provided case isn't valid")
		return
	}

	fmt.Printf("searchExp: %v\n", searchExp)

	idx := searchExp.FindIndex(*pdfContent)

	fmt.Printf("Index found: %v\n", idx)

	if len(idx) == 0 {
		respondWithError(w, 500, "Didn't find case")
		return
	}

	type successResponse struct {
		Index   string `json:"index"`
		Content string `json:"content"`
	}

	contentAsStr := string(*pdfContent)

	w.WriteHeader(200)
	w.Write([]byte(contentAsStr)[idx[0]:idx[1]])
	// respondWithJSON(w, 200, successResponse{
	// 	Index:   fmt.Sprint(idx[0]),
	// 	Content: contentAsStr[idx[0]:idx[1]],
	// })
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
