package main

import (
	"flag"
	"fmt"

	"log"
	"os"

	"github.com/hellodeveye/mindmapgen/internal/drawer"
	"github.com/hellodeveye/mindmapgen/internal/parser"
)

func main() {
	// Define command-line flags
	inputFile := flag.String("i", "input.txt", "Path to the input text file (e.g., -i input.md)")
	outputFile := flag.String("o", "output.png", "Path for the output PNG image (e.g., -o mindmap.png)")

	// Customize usage message (optional, but good practice)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generates a mind map PNG from a text file.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  %s -i my_notes.txt -o my_mindmap.png\n", os.Args[0])
	}

	// Parse the flags
	flag.Parse()

	// Read input file using os.ReadFile
	content, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file '%s': %v", *inputFile, err)
	}

	// Parse the content
	root, err := parser.Parse(string(content))
	if err != nil {
		log.Fatalf("Failed to parse input file '%s': %v", *inputFile, err)
	}

	// Apply layout (if needed - depends on whether layout is done in Draw)
	// layout.Layout(root) // Assuming layout logic is handled within Draw or not needed separately anymore

	// Draw the mind map
	err = drawer.Draw(root, *outputFile)
	if err != nil {
		log.Fatalf("Failed to draw mind map to '%s': %v", *outputFile, err)
	}

	log.Printf("Successfully generated mind map at %s from %s", *outputFile, *inputFile)
}
