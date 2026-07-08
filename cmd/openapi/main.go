package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Authula/authula/openapiexport"
)

func main() {
	outputPath := flag.String("output", "openapi.json", "Path to write the OpenAPI spec file")
	format := flag.String("format", "json", "Output format: json or yaml")
	openAPIVersion := flag.String("openapi-version", "3.1.0", "OpenAPI spec version: 3.0.3 or 3.1.0")
	flag.Parse()

	if err := openapiexport.ExportSpecToFileWithVersion(*outputPath, *format, *openAPIVersion, nil); err != nil {
		log.Fatal("Error exporting OpenAPI spec: ", err)
	}

	fmt.Printf("OpenAPI spec written to %s (%s)\n", *outputPath, *format)
}
