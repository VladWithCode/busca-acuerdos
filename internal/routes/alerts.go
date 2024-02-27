package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vladwithcode/juzgados/internal/whatsapp"
)

func SendTestMessage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		headerVars []whatsapp.TemplateVar
		bodyVars   []whatsapp.TemplateVar
	)

	y, m, d := time.Now().Date()

	var (
		dStr string
		mStr string
	)

	if d < 10 {
		dStr = fmt.Sprintf("0%d", d)
	} else {
		dStr = fmt.Sprintf("%d", d)
	}

	if m < 10 {
		mStr = fmt.Sprintf("0%d", m)
	} else {
		mStr = fmt.Sprintf("%d", m)
	}

	headerVars = append(headerVars, whatsapp.TemplateVar{
		"type": "document",
		"document": struct {
			Link     string `json:"link"`
			Filename string `json:"filename"`
		}{Link: "https://www.postgresql.org/files/documentation/pdf/16/postgresql-16-US.pdf", Filename: "reporte-.pdf"},
	})

	bodyVars = append(bodyVars, whatsapp.TemplateVar{
		"type": "text",
		"text": "Jairo Rangel",
	})
	bodyVars = append(bodyVars, whatsapp.TemplateVar{
		"type": "date_time",
		"date_time": struct {
			FallbackValue string `json:"fallback_value"`
		}{
			FallbackValue: fmt.Sprintf("%v-%v-%v", y, mStr, dStr),
		},
	})

	err := whatsapp.SendTemplateMessage("+526183188452", whatsapp.TemplateData{
		TemplateName: "report_file",
		HeaderVars:   headerVars,
		BodyVars:     bodyVars,
	})

	w.Header().Add("Content-Type", "text/html")

	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("<p>Error</p>"))

		return
	}

	w.WriteHeader(200)
	w.Write([]byte("<p>Success</p>"))
}
