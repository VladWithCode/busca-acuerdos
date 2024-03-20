package auth

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/db"
)

type Auth struct {
	Id                 string
	Username           string
	SubscriptionActive bool
}

type AuthedHandler func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, auth *Auth)

type AuthClaims struct {
	Id                 string
	Username           string
	SubscriptionActive bool

	jwt.MapClaims
}

func CreateToken(user *db.User) (string, error) {
	var (
		t *jwt.Token
		k = os.Getenv("JWT_SECRET")
	)

	t = jwt.NewWithClaims(jwt.SigningMethodHS256, AuthClaims{
		user.Id,
		user.Username,
		user.SubscriptionActive,
		jwt.MapClaims{},
	})

	return t.SignedString([]byte(k))
}

func ParseToken(tokenStr string) (*jwt.Token, error) {
	var (
		t *jwt.Token
		k = os.Getenv("JWT_SECRET")
	)

	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method %v", t.Header["alg"])
		}

		return []byte(k), nil
	})

	if err != nil {
		return nil, err
	}

	return t, nil
}

func WithAuthMiddleware(next AuthedHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		cookieToken, err := r.Cookie("auth_token")

		if err != nil {
			RejectUnauthenticated(w, r, nil, "No se encontro token")
			return
		}

		tokenStr := strings.Split(cookieToken.String(), "=")

		if len(tokenStr) < 2 {
			RejectUnauthenticated(w, r, nil, "Token invalido")
			return
		}

		t, err := ParseToken(tokenStr[1])

		if err != nil {
			RejectUnauthenticated(w, r, nil, "Sesion Token invalido")
			return
		}

		if claims, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
			var (
				id, ok1                 = claims["Id"].(string)
				username, ok2           = claims["Username"].(string)
				subscriptionActive, ok3 = claims["SubscriptionActive"].(bool)
			)

			if !ok1 || !ok2 || !ok3 {
				RejectUnauthenticated(w, r, nil, "Sesion Token invalido")
				return
			}

			next(w, r, ps, &Auth{
				Id:                 id,
				Username:           username,
				SubscriptionActive: subscriptionActive,
			})
		} else {
			RejectUnauthenticated(w, r, nil, "Sesion Token invalido")
		}
	}

}

func RejectUnauthenticated(w http.ResponseWriter, r *http.Request, _ httprouter.Params, reason string) {
	fmt.Printf("reason: %v\n", reason)

	w.Header().Add("Content-Type", "text/html")
	templ, err := template.ParseFiles("web/templates/layout.html", "web/templates/sign-in.html")

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		w.Write([]byte("<p>Ocurrió un error inesperado</p>"))
	}

	w.WriteHeader(401)
	w.Header().Add("HX-Location", "/iniciar-sesion")
	templ.Execute(w, nil)
}
