package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal"
	"github.com/vladwithcode/juzgados/internal/auth"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/mailing"
)

func RegisterUserRoutes(router *httprouter.Router) {
	router.GET("/dashboard", auth.WithAuthMiddleware(RenderDashboard))
	router.GET("/iniciar-sesion", auth.CheckAuthMiddleware(RenderSignin))
	router.GET("/registrarse", auth.CheckAuthMiddleware(RenderSignup))
	router.GET("/sign-out", auth.CheckAuthMiddleware(SignOutUser))
	router.POST("/sign-up", SignUpUser)
	router.POST("/sign-in", SignInUser)

	router.GET("/api/users/verification", RenderVerification)
	router.POST("/api/user", CreateUser)
}

func RenderDashboard(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	user, err := db.GetUserByUsername(auth.Username)

	if err != nil {
		respondWithError(w, 500, "Ocurrio un error con el servidor")
		return
	}

	alerts, err := db.FindAlertsByUser(auth.Id, false)

	if err != nil {
		fmt.Printf("[Alert Find Err]: %v\n", err)
	}

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
		"FormatDate": internal.FormatDate,
	}).ParseFiles("web/templates/layout.html", "web/templates/alert-card.html", "web/templates/dashboard.html")

	if err != nil {
		fmt.Printf("Parse err: %v\n", err)
		respondWithError(w, 500, "Ocurrio un error inseperado")
		return
	}

	data := struct {
		User   *db.User
		Alerts []*db.Alert
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

func RenderSignup(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	if auth.Id != "" {
		http.Redirect(w, r, "/dashboard", 302)
		return
	}

	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/sign-up.html")

	if err != nil {
		fmt.Printf("Parse err: %v\n", err)
		respondWithError(w, 500, "Server Error")
		return
	}

	templ.Execute(w, nil)
}

func RenderSignin(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	if auth.Id != "" {
		http.Redirect(w, r, "/dashboard", 302)
		return
	}

	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/sign-in.html")

	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Server Error")
		return
	}

	data := map[string]any{
		"User": auth,
	}

	templ.Execute(w, data)
}

func RenderVerification(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	code := r.URL.Query().Get("code")
	userId := r.URL.Query().Get("userId")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	tx, conn, err := db.GetTxAndPool(ctx)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}
	defer conn.Release()
	defer tx.Rollback(ctx)

	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/email-verification.html")

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	data := map[string]any{
		"VerificationSuccess": false,
		"VerificationMsg":     "",
		"VerificationErrMsg":  "",
	}
	cId, _ := uuid.Parse(code)
	uId, _ := uuid.Parse(userId)

	otl, err := db.TxFindUserOTLinkByCode(ctx, tx, cId, uId)

	if err != nil {
		var target *db.NonExistentOTLError
		if errors.As(err, &target) {
			data["VerificationErrMsg"] = "El codigo proporcionado no existe o no esta relacionado al usuario con el que se intento verificar"
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		err = templ.Execute(w, data)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Ocurrio un error inesperado"))
		}
		return
	}

	if !otl.CheckExpiration() {
		data["VerificationErrMsg"] = fmt.Sprintf("El codigo proporcionado expiró el %s", internal.FormatTimestampToString(otl.ExpiresAt))
		w.WriteHeader(400)
		err = templ.Execute(w, data)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Ocurrio un error inesperado"))
		}
		return
	}

	if otl.Used {
		data["VerificationErrMsg"] = "El código ya fue usado"
		w.WriteHeader(400)
		err = templ.Execute(w, data)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Ocurrio un error inesperado"))
		}
		return
	}

	err = db.TxMarkOTLinkAsUsed(ctx, tx, cId, uId)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	err = db.TxVerifyUserEmail(ctx, tx, userId)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("Commit err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	data["VerificationSuccess"] = true

	err = templ.Execute(w, data)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}
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

	_, err = db.CreateUser(&data)

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

func SignUpUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := r.ParseForm()

	if err != nil {
		fmt.Printf("ParseForm err: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("HX-Reswap", "beforebegin")
		w.Header().Set("HX-Reselect", "#form-invalid-feedback")
		w.Header().Set("HX-Retarget", "[data-form-submit-btn]")
		w.WriteHeader(400)
		w.Write([]byte("<p class=\"text-secondary-500 font-medium\" id=\"form-invalid-feedback\" data-form-invalid-tag=\"form\">El formulario contiene información inválida</p>"))
		return
	}

	formTempl, err := template.New("layout.html").ParseFiles("web/templates/layout.html", "web/templates/sign-up.html")

	if err != nil {
		fmt.Printf("Parse err: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("HX-Redirect", "/500")
		w.WriteHeader(500)
		return
	}

	pw := r.Form.Get("password")
	confirmPw := r.Form.Get("confirmPassword")

	if pw != confirmPw {
		w.Header().Set("HX-Reselect", "#form-confirm-password-group")
		w.Header().Set("HX-Retarget", "#form-confirm-password-group")
		w.Header().Set("HX-Reswap", "outerHTML")
		w.WriteHeader(400)
		formTempl.Execute(w, map[string]bool{
			"PasswordNoMatch": true,
		})
		return
	}

	id, _ := uuid.NewV7()

	user := &db.User{
		Id:       id.String(),
		Name:     r.Form.Get("name"),
		Lastname: r.Form.Get("lastname"),
		Username: r.Form.Get("username"),
		Email:    r.Form.Get("email"),
		Password: pw,
	}

	_, err = db.CreateUser(user)

	if err != nil {
		var PgErr *pgconn.PgError
		isPgErr := errors.As(err, &PgErr)
		if isPgErr {
			//fmt.Printf("PgErr: %+v\n", *PgErr)
			templ, err := template.New("blocks.html").ParseFiles("web/templates/blocks.html")

			if err == nil && PgErr.Code == "23505" {
				w.WriteHeader(400)
				templ.ExecuteTemplate(w, "error-card", map[string]string{
					"Message":   "Usuario existente",
					"BtnLabel":  "Aceptar",
					"ErrorCode": PgErr.Code,
				})
				return
			}
		}

		fmt.Printf("Create err: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	userUUID, err := uuid.Parse(user.Id)
	if err != nil {
		fmt.Printf("Parse id err: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	otl, err := db.CreateVerifyOTL(userUUID)

	if err != nil {
		fmt.Printf("Create OTL err: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	if err := mailing.SendVerificationMail(user.Email, otl); err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
		return
	}

	w.Header().Set("HX-Reswap", "outerHTML")
	err = formTempl.ExecuteTemplate(w, "signup-success", map[string]string{
		"RegistrationEmail": user.Email,
	})

	if err != nil {
		fmt.Printf("Execute succ templ err: %v\n", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("Ocurrio un error inesperado"))
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
		Expires:  time.Now().Add(6 * time.Hour),
		HttpOnly: true,
		//Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, jwtCookie)
	w.Header().Add("HX-Location", "/dashboard")

	respondWithJSON(w, 204, user)
}

func SignOutUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params, auth *auth.Auth) {
	if auth.Id == "" {
		http.Redirect(w, r, "/", 401)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		// Secure: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Add("HX-Location", "/")
	http.Redirect(w, r, "/", 302)
}
