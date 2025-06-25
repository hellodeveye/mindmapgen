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
	themeName := flag.String("theme", "default", "Theme to use for the mind map (e.g., default, dark, business)")

	// Customize usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generates a mind map PNG from a text file with customizable themes.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -i input.txt -o output.png\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i input.txt -o output.png -theme dark\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -raw \"mindmap\\n  root((Main Topic))\\n    Subtopic\" -theme business\n", os.Args[0])
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

	if len(content) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No input provided. Use -i for file input or -raw for direct text input.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the content
	root, err := parser.Parse(string(content))
	if err != nil {
		log.Fatalf("Failed to parse input: %v", err)
	}

	if *b64 {
		w := base64.NewEncoder(base64.StdEncoding, os.Stdout)
		defer w.Close()
		err := drawer.DrawWithTheme(root, w, *themeName)
		if err != nil {
			log.Fatalf("Failed to draw mind map: %v", err)
		}
		return
	}

	f, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file '%s': %v", *outputFile, err)
	}
	defer f.Close()

	// Draw the mind map with specified theme
	err = drawer.DrawWithTheme(root, f, *themeName)
	if err != nil {
		log.Fatalf("Failed to draw mind map: %v", err)
	}

	log.Printf("Successfully generated mind map at %s using theme '%s'", *outputFile, *themeName)
}
