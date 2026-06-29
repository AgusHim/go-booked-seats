//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
)

func main() {
	for _, path := range os.Args[1:] {
		f, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		stat, _ := f.Stat()
		r, err := pdf.NewReader(f, stat.Size())
		if err != nil {
			panic(err)
		}
		fmt.Println("FILE", path)
		for pageNum := 1; pageNum <= r.NumPage(); pageNum++ {
			page := r.Page(pageNum)
			xobjs := page.Resources().Key("XObject")
			fmt.Println(" page", pageNum, "xobject keys", xobjs.Keys())
			for _, key := range xobjs.Keys() {
				x := xobjs.Key(key)
				fmt.Printf("  %s kind=%v subtype=%s filter=%s width=%d height=%d bpc=%d colorspace=%s decodeparms=%s\n",
					key, x.Kind(), x.Key("Subtype").Name(), x.Key("Filter").Name(),
					x.Key("Width").Int64(), x.Key("Height").Int64(), x.Key("BitsPerComponent").Int64(),
					x.Key("ColorSpace").String(), x.Key("DecodeParms").String())
			}
		}
		_ = f.Close()
	}
}
