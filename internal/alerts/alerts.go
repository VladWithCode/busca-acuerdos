package alerts

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/vladwithcode/juzgados/internal"
	"github.com/vladwithcode/juzgados/internal/db"
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

func GenReportPdfWithData(userData db.AutoReportUser) (docPath string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	templ, err := template.New("layout.html").Funcs(template.FuncMap{
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
		"GetNature": func(nc string) string {
			return internal.CodesMap[nc]
		},
	}).ParseFiles("web/templates/reports/layout.html", "web/templates/reports/alert-report.html", "web/templates/reports/css.html")

	if err != nil {
		return
	}

	file, err := os.CreateTemp("/var/tmp", "report-*.html")

	if err != nil {
		return
	}

	if err = os.Chmod(file.Name(), 0644); err != nil {
		fmt.Println("Error changing file permissions", err)
		return
	}
	defer os.Remove(file.Name())

	err = templ.Execute(file, userData)

	if err != nil {
		return
	}

	dirPath := path.Clean(fmt.Sprintf("web/static/reports/%v/report.pdf", userData.Id))
	printFlag := fmt.Sprintf(fmt.Sprintf("--print-to-pdf=%v", dirPath))

	chromeCmd := exec.CommandContext(
		ctx,
		"chromium",
		"--headless=new",
		"--disable-gpu",
		"--no-pdf-header-footer",
		printFlag,
		file.Name(),
	)
	chromeCmd.Stdout = os.Stdout
	chromeCmd.Stderr = os.Stderr

	if err = chromeCmd.Run(); err != nil {
		fmt.Println("Error executing cmd: ", err)
		return
	}

	docPath = fmt.Sprintf("/reports/%v/report.pdf", userData.Id)

	return
}
