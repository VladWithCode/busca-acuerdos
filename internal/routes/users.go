package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	user, err := db.GetUserByUsername(auth.Username)

	if err != nil {
		respondWithError(w, 500, "Ocurrio un error con el servidor")
		return
	}

	alerts, err := db.FindAlertsByUser(auth.Id)

	if err != nil {
		fmt.Printf("[Alert Find Err]: %v\n", err)
	}

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
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
	}).ParseFiles("web/templates/layout.html", "web/templates/dashboard.html")

	if err != nil {
		fmt.Printf("err: %v\n", err)
		respondWithError(w, 500, "Ocurrio un error inseperado")
		return
	}

	data := struct {
		User   *db.User
		Alerts *[]db.Alert
	}{
		User:   user,
		Alerts: alerts,
	}

	err = templ.Execute(
		w,
		data,
	)

	if err != nil {
		fmt.Printf("[Execute Error]: %v\n", err)

		respondWithError(w, 500, "Ocurrio un error inesperado")
		return
	}
}

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

	dirPath := fmt.Sprintf("web/static/reports/%v", data.Id)
	err = os.Mkdir(dirPath, 0666)

	if err != nil {
		fmt.Printf("MkDir Err: %v", err)
	}
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
