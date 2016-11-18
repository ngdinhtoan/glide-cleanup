package main

import (
	"flag"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/glide/cfg"
	"github.com/Masterminds/glide/msg"
)

var (
	glideYaml = "glide.yaml"
	logger    = msg.NewMessenger()
)

func init() {
	flag.StringVar(&glideYaml, "yaml", "glide.yaml", "Set a YAML configuration file")
	flag.BoolVar(&logger.IsVerbose, "verbose", false, "Print more verbose informational messages")
	flag.BoolVar(&logger.IsDebugging, "debug", false, "Print debug verbose informational messages")
	flag.BoolVar(&logger.Quiet, "quiet", false, "Quiet (no info or debug messages)")
}

func main() {
	flag.Parse()

	if logger.IsDebugging {
		logger.IsVerbose = true
	}

	// load package from glide.yml config
	logger.Verbose("Loading Glide config from %s...", glideYaml)
	glideConfig, err := loadGlideConfig(glideYaml)
	if err != nil {
		logger.Die("Could not load Glide configuration from glide.yaml file: %v", err)
	}

	logger.Verbose("Collecting imported packages...")
	importPkgs := make(map[string]interface{})
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if path == "vendor" {
				return filepath.SkipDir
			}

			if path == "Godeps" {
				return filepath.SkipDir
			}

			if path != "." && strings.HasPrefix(path, ".") {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		logger.Debug("--> getting import package in %q", path)
		pkgs, err := getImports(path)
		if err == nil {
			for _, pkg := range pkgs {
				importPkgs[pkg] = nil
			}
		} else {
			logger.Err("Error when get import package for file %v: %v", path, err)
		}

		return nil
	})

	if err != nil {
		logger.Die(err.Error())
	}

	logger.Verbose("Checking unused packages...")
	unusedPkgs := make(map[string]interface{})
gi:
	for _, dep := range glideConfig.Imports {
		if _, found := importPkgs[dep.Name]; found {
			delete(importPkgs, dep.Name)
			continue
		}

		// todo: if subpackages is defined, need to check it all

		for pkg := range importPkgs {
			if strings.HasPrefix(pkg, dep.Name) {
				continue gi
			}
		}

		logger.Debug("--> package %q is not used", dep.Name)
		unusedPkgs[dep.Name] = nil
	}

	if len(unusedPkgs) == 0 {
		logger.Info("Well done! All packages are needed.")
		os.Exit(0)
	}

	logger.Verbose("Removing unused packages...")
	for pkg := range unusedPkgs {
		glideConfig.Imports = glideConfig.Imports.Remove(pkg)
	}

	if err = glideConfig.WriteFile(glideYaml); err != nil {
		logger.Die("Error while write Glide config to file back: %v", err)
	}

	logger.Info("New Glide config has been updated with removing unused packages.")
}

func loadGlideConfig(file string) (config *cfg.Config, err error) {
	var yml []byte

	if yml, err = ioutil.ReadFile(file); err != nil {
		return nil, err
	}

	return cfg.ConfigFromYaml(yml)
}

func getImports(file string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	if f.Imports == nil || len(f.Imports) == 0 {
		return []string{}, nil
	}

	pkgs := make([]string, 0, len(f.Imports))
	for _, importSpec := range f.Imports {
		if importSpec.Path == nil {
			continue
		}
		// todo: check if package is a Go built-in package
		// todo: check if package is a sub-package of checking package
		pkg := strings.Trim(importSpec.Path.Value, `"`)
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}
