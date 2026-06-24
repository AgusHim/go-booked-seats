package main

import (
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
)

func main() {
	f, err := os.Open("../Agus Himawan (WTI33M0S).pdf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	stat, _ := f.Stat()
	pdfReader, err := pdf.NewReader(f, stat.Size())
	if err != nil {
		panic(err)
	}

	page := pdfReader.Page(1)
	text, err := page.GetPlainText(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("--- EXTRACTED TEXT ---")
	fmt.Println(text)
	fmt.Println("----------------------")
}
