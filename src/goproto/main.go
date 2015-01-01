package main

import (
	"flag"
	"generator"
	"io/ioutil"
	"os"
)

func main() {
	src := flag.String("src", "", "set protocol file path")
	dest := flag.String("dest", "", "protocol code's file")
	flag.Parse()
	if len(*src) != 0 && len(*dest) != 0 {
		data, err := generator.Generate(*src)
		if err != nil {
			println(err.Error())
			return
		}
		ioutil.WriteFile(*dest, data, os.ModePerm)
		println("Complete!")
	}
}
