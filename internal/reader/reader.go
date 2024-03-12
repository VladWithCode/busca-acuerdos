package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

const fileUrl = "http://tsjdgo.gob.mx/Recursos/images/flash/ListasAcuerdos/%v/%v.pdf"

func GenRegExp(caseId string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf(`(?m)^(\d[^\n]*%v\s[^\n][^\d]*)$`, caseId))
}

func GetFile(date, caseType string) (pdfData []byte, err error) {
	fetchUrl := fmt.Sprintf(fileUrl, date, caseType)

	response, err := http.Get(fetchUrl)

	if err != nil {
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

func PipeLargeFile(fileData []byte) (*[]byte, error) {
	pr, pw, _ := os.Pipe()
	outPr, outPw, _ := os.Pipe()
	outputBuf := new(bytes.Buffer)

	pttCmd := exec.Command("pdftotext", "-layout", "-", "-")

	pttCmd.Stdin = pr
	pttCmd.Stdout = outPw

	if err := pttCmd.Start(); err != nil {
		fmt.Println("Error starting: ", err)
		return nil, err
	}

	go func() {
		defer pw.Close()

		pw.Write(fileData)
	}()

	go func() {
		defer outPr.Close()

		outputBuf.ReadFrom(outPr)
	}()

	if err := pttCmd.Wait(); err != nil {
		return nil, err
	}

	data := outputBuf.Bytes()

	return &data, nil
}

// Uses poppler-utils' pdftotext to parse the pdf
func ParseFile(fileData []byte) (*[]byte, error) {
	if len(fileData) > 65_000 {
		return PipeLargeFile(fileData)
	}

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
