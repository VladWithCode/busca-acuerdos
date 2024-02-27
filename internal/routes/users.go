package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
)

func SignInHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/sign-in.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
	}

	templ.Execute(w, nil)
}

func CreateUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data := db.User{}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err := decoder.Decode(&data)

	if err != nil {
		fmt.Printf("Malformed json data: %v\n", err)
		respondWithError(w, 400, "La información proporcionada es inválida")
		return
	}

	_, err = db.CreateUser(data.Id, data.Name, data.Lastname, data.Username, data.Email, data.Phone, data.Password, data.SubscriptionActive)

	if err != nil {
		fmt.Println(err)

		if strings.Contains(err.Error(), "duplicate key") {
			respondWithError(w, 400, "No se pudo crear el usuario, el email y/o telefono ya estan registrados")
			return
		}

		respondWithError(w, 500, "No se pudo crear el usuario")
		return
	}

	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte("<p>Creación exitosa</p>"))
}

func SignInUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := r.ParseForm()

	if err != nil {
		fmt.Printf("ParseForm err: %v\n", err)
		respondWithError(w, 400, "La información proporcionada no es válida")
		return
	}

	data := struct {
		Username string
		Password string
	}{
		Username: r.Form.Get("username"),
		Password: r.Form.Get("password"),
	}

	user, err := db.GetUserByUsername(data.Username)

	if err != nil {
		fmt.Printf("Get Error: %v\n", err)
		respondWithError(w, 400, "Error al recuperar el usuario")
		return
	}

	err = user.ValidatePass(data.Password)

	if err != nil {
		respondWithError(w, 400, "La contraseña es inválida")
		return
	}

	user.Password = ""

	t, err := auth.CreateToken(user)

	if err != nil {
		fmt.Printf("CreateToken err: %v\n", err)
		respondWithError(w, 500, "Ocurrio un error al crear la sesión. Intente de nuevo más tarde")
		return
	}

	jwtCookie := &http.Cookie{
		Name:     "auth_token",
		Value:    t,
		Expires:  time.Now().Add(2 * time.Hour),
		HttpOnly: true,
		//Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, jwtCookie)
	w.Header().Add("HX-Location", "/dashboard")

	respondWithJSON(w, 200, user)
}
