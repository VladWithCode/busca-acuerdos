package alerts

import (
	"context"
	"fmt"
	"html/template"
	"os/exec"
	"path"
	"time"
)

type CaseData struct {
	caseId     string
	nature     string
	natureCode string
	accord     string
	accordDate time.Time
}

type ReportData struct {
	username   string
	cases      []CaseData
	reportDate time.Time
}

func RegisterAlertForCase(userId string, caseId string, natureCode string) {

}

func GenReportPdf(userId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	url := fmt.Sprintf("http://localhost:8080/api/alerts/report/%v", userId)
	dirPath := path.Clean(fmt.Sprintf("web/static/reports/%v/report.pdf", userId))
	printFlag := fmt.Sprintf(fmt.Sprintf("--print-to-pdf=%v", dirPath))

	chromeCmd := exec.CommandContext(
		ctx,
		"chromium",
		"--headless=new",
		"--disable-gpu",
		"--no-pdf-header-footer",
		printFlag,
		url,
	)

	err := chromeCmd.Run()

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/reports/%v/report.pdf", userId), nil
}

func CreateDocument(reportData *ReportData) (*template.Template, error) {
	templ, err := template.New("alert-report.html").Funcs(template.FuncMap{

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
	}).ParseFiles("web/templates/alert-report.html")

	if err != nil {
		return nil, err
	}

	return templ, err
}
