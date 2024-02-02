package reader

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

const fileUrl = "http://tsjdgo.gob.mx/Recursos/images/flash/ListasAcuerdos/%v/%v.pdf"

func GenRegExp(caseId string) (*regexp.Regexp, error) {
	id := strings.Replace(caseId, "/", `\/`, 1)
	return regexp.Compile(fmt.Sprintf(`(?m)^(\d[^\n]*%v[^\n][^\d]*)$`, id))
}

func GetFile(date, caseType string) (pdfData []byte, err error) {
	fetchUrl := fmt.Sprintf(fileUrl, date, caseType)

	response, err := http.Get(fetchUrl)

	if err != nil {
		fmt.Printf("err: %v\n", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return nil, errors.New("No se encontró documento para la fecha solicitada")
	}

	pdfData, err = io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	return pdfData, nil
}

// Uses poppler-utils' pdftotext to parse the pdf
func ParseFile(fileData []byte) (*[]byte, error) {
	// $ pdftotext [options] [PDF-File [text-file]]
	// If text-file is '-'. "-" means pipe from/to stdin/stdout
	pttCmd := exec.Command("pdftotext", "-layout", "-", "-")
	pipe, err := pttCmd.StdinPipe()

	if err != nil {
		return nil, err
	}

	_, err = pipe.Write(fileData)

	if err != nil {
		return nil, err
	}

	err = pipe.Close()

	if err != nil {
		return nil, err
	}

	output, err := pttCmd.Output()

	if err != nil {
		return nil, err
	}

	return &output, nil
}

func Reader(date, caseType string) (result *[]byte, err error) {
	pdfData, err := GetFile(date, caseType)
	if err != nil {
		return nil, err
	}

	return ParseFile(pdfData)
}
