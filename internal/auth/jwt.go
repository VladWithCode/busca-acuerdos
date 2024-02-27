package auth

import (
	"fmt"
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

	jwt.RegisteredClaims
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
		jwt.RegisteredClaims{},
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
			RejectUnauthenticated(w, r, nil, "Error al leer cookies")
			return
		}

		tokenStr := strings.Split(cookieToken.String(), "=")

		if len(tokenStr) < 2 {
			RejectUnauthenticated(w, r, nil, "Token invalido")
			return
		}

		t, err := ParseToken(tokenStr[1])

		if err != nil {
			fmt.Printf("err: %v\n", err)
			RejectUnauthenticated(w, r, nil, "Sesion Token invalido")
			return
		}

		if claims, ok := t.Claims.(AuthClaims); ok && t.Valid {
			next(w, r, ps, &Auth{
				Id:                 claims.Id,
				Username:           claims.Username,
				SubscriptionActive: claims.SubscriptionActive,
			})
		} else {
			RejectUnauthenticated(w, r, nil, "Sesion Token invalido")
		}
	}

}

func RejectUnauthenticated(w http.ResponseWriter, r *http.Request, _ httprouter.Params, reason string) {
	fmt.Printf("reason: %v\n", reason)
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(503)
	w.Write([]byte("<p>Unauthorized</p>"))
}
