package reader

import (
	"bytes"
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

const fileUrl = "http://tsjdgo.gob.mx/Recursos/images/flash/ListasAcuerdos/4122023/civ2.pdf"
const tempFilePath = "/tmp/temp_pdf.pdf"

func GenRegExp(caseId string) (*regexp.Regexp, error) {
	id := strings.Replace(caseId, "/", `\/`, 1)
	return regexp.Compile(fmt.Sprintf(`(?m)^(\d[^\n]*%v[^\n][^\d]*)$`, id))
}

func GetFile() (pdfData []byte, err error) {
	response, err := http.Get(fileUrl)

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	pdfData, err = io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	fmt.Printf("Read %v bytes from file\n", len(pdfData))

	return pdfData, nil
}

// Uses pdfcpu to get the contents of a file
func ParseFileWithLDPdf(filepath string) (*[]byte, error) {

	f, r, err := pdf.Open(tempFilePath)

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

func Reader() (result *[]byte, err error) {
	pdfData, err := GetFile()
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(tempFilePath, pdfData, 0644)

	if err != nil {
		log.Fatal(err)
	}

	return ParseFile(tempFilePath)
}
