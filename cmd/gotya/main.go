package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/GabrielNunesIT/gotya"
	"github.com/GabrielNunesIT/gotya/generator/golang"
	"github.com/GabrielNunesIT/gotya/generator/protobuf"
	"github.com/GabrielNunesIT/gotya/schema"
)

// main is the CLI tool entrypoint for parsing and compiling YANG files using gotya.
//
// Why: Providing a CLI wrapper enables gotya to act as a standalone linter/compiler
// or to be driven within Makefile and Bazel rules out of the box.
func main() {
	pathsFlag := flag.String("paths", "", "Comma-separated list of directories to search for imported/included YANG modules")
	outdirFlag := flag.String("outdir", "out", "Directory path where generated files will be written")
	formatFlag := flag.String("format", "go", "Output format for code generation (go, pb)")
	pkgFlag := flag.String("package_name", "", "Generated package name (default: device for go, gotya.device for protobuf)")
	rootFlag := flag.String("fakeroot_name", "Device", "Name of the root generated entity")
	fakeRootFlag := flag.Bool("generate_fakeroot", false, "If set to true, a fake element at the root of the data tree is generated")
	skipDepFlag := flag.Bool("skip_deprecated", false, "If set to true, skip generating code for deprecated nodes")
	skipObsFlag := flag.Bool("skip_obsolete", false, "If set to true, skip generating code for obsolete nodes")
	annotationsFlag := flag.Bool("add_annotations", false, "If set to true, add a generic metadata field to generated structs")
	gettersFlag := flag.Bool("generate_getters", false, "If set to true, generate getter methods for fields")
	settersFlag := flag.Bool("generate_setters", false, "If set to true, generate setter methods for fields")
	populateDefFlag := flag.Bool("generate_populate_default", false, "If set to true, generate PopulateDefaults method")
	orderedMapsFlag := flag.Bool("generate_ordered_maps", false, "If set to true, generates lists as slices instead of maps")
	protoCELFlag := flag.Bool("proto_cel", false, "If set to true, emit bufbuild/protovalidate CEL constraints for protobuf")

	flag.Usage = func() {
		//nolint:gosec // Usage text is inherently safe from command injection.
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <yang files...>\n\n", filepath.Clean(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	searchPaths := []string{"."} // Default to current directory
	if *pathsFlag != "" {
		searchPaths = append(searchPaths, strings.Split(*pathsFlag, ",")...)
	}

	loader := NewDirectoryLoader(searchPaths)

	var astModules []*gotya.ASTModule
	for _, file := range args {
		astMod, err := gotya.ParseFile(file)
		if err != nil {
			log.Fatalf("Error parsing file %s: %v", file, err)
		}

		if astMod.Keyword() == "submodule" {
			continue // Handled natively by compiler via include statements
		}

		astModules = append(astModules, astMod)
	}

	// 2. Compile modules
	var schemaModules []*schema.Module
	for _, astMod := range astModules {
		mod, err := loader.Load(astMod.Argument())
		if err != nil {
			log.Printf("Compilation errors in %s: %v", astMod.Argument(), err)
		}
		if mod != nil {
			schemaModules = append(schemaModules, mod)
		}
	}

	if len(schemaModules) == 0 {
		log.Fatalf("No valid schemas compiled. Exiting.")
	}

	// 3. Ensure output directory exists
	if err := os.MkdirAll(*outdirFlag, 0750); err != nil {
		log.Fatalf("Failed to create outdir %s: %v", *outdirFlag, err)
	}

	// 4. Generate target outputs
	switch *formatFlag {
	case "go":
		generateGo(schemaModules, *outdirFlag, *pkgFlag, *rootFlag, *fakeRootFlag, *skipDepFlag, *skipObsFlag, *annotationsFlag, *gettersFlag, *settersFlag, *populateDefFlag, *orderedMapsFlag)
	case "pb":
		generateProtobuf(schemaModules, *outdirFlag, *pkgFlag, *rootFlag, *fakeRootFlag, *skipDepFlag, *skipObsFlag, *annotationsFlag, *gettersFlag, *settersFlag, *populateDefFlag, *orderedMapsFlag, *protoCELFlag)
	default:
		log.Fatalf("Unknown generation format: %s", *formatFlag)
	}
}

// generateGo invokes the Golang compiler logic for generating the Device struct
func generateGo(modules []*schema.Module, outDir, pkgName, rootName string, genFakeroot, skipDep, skipObs, addAnn, genGetters, genSetters, genPopDef, genOrdMaps bool) {
	outPath := filepath.Clean(filepath.Join(outDir, "device.go"))
	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("Failed to create Go output file %s: %v", outPath, err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	if pkgName == "" {
		pkgName = "device"
	}

	gen := golang.New(&golang.Options{
		PackageName:             pkgName,
		RootName:                rootName,
		GenerateFakeroot:        genFakeroot,
		SkipDeprecated:          skipDep,
		SkipObsolete:            skipObs,
		AddAnnotations:          addAnn,
		GenerateGetters:         genGetters,
		GenerateSetters:         genSetters,
		GeneratePopulateDefault: genPopDef,
		GenerateOrderedMaps:     genOrdMaps,
	})
	if err := gen.GenerateDevice(modules, outFile); err != nil {
		log.Fatalf("Golang generator failed: %v", err)
	}
	log.Printf("Created output node: %s", outPath)
}

// generateProtobuf invokes the Protobuf compiler logic to emit .proto targets natively
func generateProtobuf(modules []*schema.Module, outDir, pkgName, rootName string, genFakeroot, skipDep, skipObs, addAnn, genGetters, genSetters, genPopDef, genOrdMaps, celVal bool) {
	outPath := filepath.Clean(filepath.Join(outDir, "device.proto"))
	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("Failed to create Protobuf output file %s: %v", outPath, err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	if pkgName == "" {
		pkgName = "gotya.device"
	}

	gen := protobuf.New(&protobuf.Options{
		PackageName:             pkgName,
		RootName:                rootName,
		GenerateFakeroot:        genFakeroot,
		SkipDeprecated:          skipDep,
		SkipObsolete:            skipObs,
		AddAnnotations:          addAnn,
		GenerateGetters:         genGetters,
		GenerateSetters:         genSetters,
		GeneratePopulateDefault: genPopDef,
		GenerateOrderedMaps:     genOrdMaps,
		GenerateCELValidation:   celVal,
	})
	if err := gen.GenerateDevice(modules, outFile); err != nil {
		log.Fatalf("Protobuf generator failed: %v", err)
	}
	log.Printf("Created output node: %s", outPath)
}
