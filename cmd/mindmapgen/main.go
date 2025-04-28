package main

import (
	"encoding/base64"
	"flag"
	"fmt"

	"log"
	"os"

	"github.com/hellodeveye/mindmapgen/internal/drawer"
	"github.com/hellodeveye/mindmapgen/internal/parser"
)

func main() {
	// Define command-line flags
	inputFile := flag.String("i", "", "Path to the input text file (e.g., -i input.md)")
	outputFile := flag.String("o", "output.png", "Path for the output PNG image (e.g., -o mindmap.png)")
	b64 := flag.Bool("b", false, "Print the output to stdout as base64 encoded string")
	rawStr := flag.String("raw", "", "Parse raw content to mind map")

	// Customize usage message (optional, but good practice)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generates a mind map PNG from a text file.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  %s -i input.txt -o output.png\n", os.Args[0])
	}

	// Parse the flags
	flag.Parse()

	var content []byte
	// Read input file using os.ReadFile
	if *inputFile != "" {
		c, err := os.ReadFile(*inputFile)
		if err != nil {
			log.Fatalf("Failed to read input file '%s': %v", *inputFile, err)
		}
		content = c
	}

	if *rawStr != "" {
		content = []byte(*rawStr)
	}

	// Parse the content
	root, err := parser.Parse(string(content))
	if err != nil {
		log.Fatalf("Failed to parse input file '%s': %v", *inputFile, err)
	}

	if *b64 {
		w := base64.NewEncoder(base64.StdEncoding, os.Stdout)
		defer w.Close()
		err := drawer.Draw(root, w)
		if err != nil {
			log.Fatalf("Failed to draw mind map to '%s': %v", *outputFile, err)
		}
		return
	}

	f, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file '%s': %v", *outputFile, err)
	}
	defer f.Close()

	// Draw the mind map
	err = drawer.Draw(root, f)
	if err != nil {
		log.Fatalf("Failed to draw mind map to '%s': %v", *outputFile, err)
	}

	log.Printf("Successfully generated mind map at %s from %s", *outputFile, *inputFile)
}
