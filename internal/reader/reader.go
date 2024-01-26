package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

const fileUrl = "http://tsjdgo.gob.mx/Recursos/images/flash/ListasAcuerdos/%v/%v.pdf"
const TempFilePath = "/tmp/temp_pdf.pdf"

func GenRegExp(caseId string) (*regexp.Regexp, error) {
	id := strings.Replace(caseId, "/", `\/`, 1)
	return regexp.Compile(fmt.Sprintf(`(?m)^(\d[^\n]*%v[^\n][^\d]*)$`, id))
}

func GetFile(date, caseType string) (pdfData []byte, err error) {
	response, err := http.Get(fmt.Sprintf(fileUrl, date, caseType))

	if err != nil {
		fmt.Printf("err: %v\n", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return nil, errors.New("No existe registro para la fecha solicitada")
	}

	pdfData, err = io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	fmt.Printf("Read %v bytes from file\n", len(pdfData))

	return pdfData, nil
}

// Uses pdfcpu to get the contents of a file
func ParseFileWithLDPdf(filepath string) (*[]byte, error) {

	f, r, err := pdf.Open(TempFilePath)

	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()

	if err != nil {
		return nil, err
	}

	pageRows, err := r.Page(0).GetTextByRow()
	if err != nil {
		return nil, err
	}

	for i, pageRow := range pageRows {
		fmt.Printf("%d %v", i, pageRow.Content)
	}

	buf.ReadFrom(b)

	bytes := buf.Bytes()

	return &bytes, nil
}

// Uses poppler-utils' pdftotext to parse the pdf
func ParseFile(filepath string) (*[]byte, error) {
	// $ pdftotext [options] [PDF-File [text-file]]
	// If text-file is '-', the text is sent to stdout (needs clarification on what '-' means)
	output, err := exec.Command("pdftotext", "-layout", filepath, "-").Output()

	if err != nil {
		return nil, err
	}

	// contentAsString := string(output)

	// rows := strings.Split(contentAsString, "\n")

	// for idx, row := range rows[:71] {
	// 	fmt.Printf("%d %v\n", idx, row)
	// }

	return &output, nil
}

func Reader(date, caseType string) (result *[]byte, err error) {
	pdfData, err := GetFile(date, caseType)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(TempFilePath, pdfData, 0644)

	if err != nil {
		log.Fatal(err)
	}

	return ParseFile(TempFilePath)
}
