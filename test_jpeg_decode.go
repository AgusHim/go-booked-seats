//go:build ignore

package main

import (
	"fmt"
	"image/jpeg"

	"github.com/ledongthuc/pdf"
)

func main() {
	f, r, err := pdf.Open("../Receipt Agus Himawan - #DTO8A8N4NYS.pdf")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer f.Close()

	page := r.Page(1)
	xobjects := page.Resources().Key("XObject")
	for _, key := range xobjects.Keys() {
		xobject := xobjects.Key(key)
		if xobject.Kind() != pdf.Stream || xobject.Key("Subtype").Name() != "Image" {
			continue
		}

		filter := xobject.Key("Filter").Name()
		if filter == "" && xobject.Key("Filter").Kind() == pdf.Array {
			filter = xobject.Key("Filter").Index(0).Name()
		}

		if filter == "DCTDecode" {
			fmt.Println("Found DCTDecode image:", key)
			reader := xobject.Reader()
			img, err := jpeg.Decode(reader)
			if err != nil {
				fmt.Println("Decode error:", err)
			} else {
				fmt.Printf("Decoded successfully! Size: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
			}
			return
		}
	}
}
