package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotya/gotya/adapters/generator/golang"
	"github.com/gotya/gotya/adapters/parser/lexer"
	"github.com/gotya/gotya/adapters/parser/parser"
	"github.com/gotya/gotya/domain/schema"
	"github.com/gotya/gotya/usecases/compiler"
)

func main() {
	yangsDir := filepath.Join("..", "..", "test", "assets", "yangs")

	// Read all .yang files in the directory
	files, err := os.ReadDir(yangsDir)
	if err != nil {
		panic(err)
	}

	var modules []*schema.Module
	opts := &compiler.Options{}
	comp := compiler.New(opts)

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yang" {
			continue
		}

		path := filepath.Join(yangsDir, file.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Parsing %s\n", file.Name())

		l := lexer.New(string(content))
		p := parser.New(l)
		astMod := p.ParseModule()
		if astMod == nil {
			fmt.Printf("Failed to parse module AST for file %s\n", file.Name())
			continue
		}

		schemaMod, err := comp.Compile(astMod)
		if err != nil {
			fmt.Printf("Module %s compiled with errors: %v\n", file.Name(), err)
		}
		if schemaMod != nil {
			modules = append(modules, schemaMod)
		}
	}

	fmt.Printf("Parsed %d modules out of %d\n", len(modules), len(files))

	gen := golang.New("device")
	var buf bytes.Buffer
	err = gen.GenerateDevice(modules, &buf)
	if err != nil {
		panic(err)
	}

	outDir := filepath.Join("..", "test", "out")
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		panic(err)
	}

	outPath := filepath.Join(outDir, "device.go")
	err = os.WriteFile(outPath, buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Device struct successfully written to %s\n", outPath)
}
