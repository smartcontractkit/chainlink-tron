package main

// Generates a Go struct template from a json source

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// convert snake_case to TitleCase
func titlecase(s string) string {
	words := strings.Split(s, "_")
	for ix, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[ix] = string(runes)
	}
	return strings.Join(words, "")
}

// print a field with indent and snake_case converted to TitleCase
func printField(name string, value string, indent int) {
	var output string
	if len(name) < 1 {
		output = strings.Repeat("  ", indent) + value + ","
	} else {
		output = strings.Repeat("  ", indent) + titlecase(name) + ": " + value + ","
	}
	fmt.Println(output)
}

func printHeader(name string, indent int) {
	if len(name) < 1 {
		return
	}
	output := strings.Repeat("  ", indent) + titlecase(name) + ": "
	fmt.Println(output)
}

func printRaw(text string, indent int) {
	output := strings.Repeat("  ", indent) + text
	fmt.Println(output)
}

// forwards maps to map handling and arrays to array handling
func determineType(name string, i interface{}, indent int) {
	if maptype, ok := i.(map[string]interface{}); ok {
		printHeader(name, indent)
		listMap(maptype, indent+1)
	} else if arrtype, ok := i.([]interface{}); ok {
		printHeader(name, indent)
		listArray(arrtype, indent+1)
	} else {
		switch value := i.(type) {
		case string:
			printField(name, "\""+value+"\"", indent)
		case float64:
			valuestr := strconv.FormatFloat(value, 'f', -1, 64)
			printField(name, valuestr, indent)
		case bool:
			valuestr := fmt.Sprintf("%v", value)
			printField(name, valuestr, indent)
		default:
			valuestr := fmt.Sprintf("unrecognized type %T", i)
			printField(name, valuestr, indent)

		}
	}
}

// processes a json array
func listArray(a []interface{}, indent int) {
	printRaw("[]{", indent)
	for _, value := range a {
		determineType("", value, indent)
	}
	printRaw("},", indent)
}

// processes a json object ie. a Map in Go
func listMap(f map[string]interface{}, indent int) {
	printRaw("{", indent)
	for name, value := range f {
		//fmt.Printf("name: %s, value type: %T\n", name, value)
		determineType(name, value, indent)
	}
	printRaw("},", indent)
}

func main() {

	if len(os.Args) < 2 {
		log.Fatalf("missing json source file name")
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("error opening json source: %v", err)
	}

	src, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("error reading json source: %v", err)
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(src, &fields); err != nil {
		log.Fatalf("error unmarshaling source json: %v", err)
	}

	listMap(fields, 0)
}
