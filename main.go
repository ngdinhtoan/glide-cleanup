package main

import (
	"flag"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/glide/action"
	"github.com/Masterminds/glide/cfg"
	"github.com/Masterminds/glide/msg"
	gpath "github.com/Masterminds/glide/path"
)

var (
	glideYaml  = gpath.DefaultGlideFile
	argDebug   = false
	argVerbose = false
	argQuiet   = false
)

func init() {
	flag.StringVar(&glideYaml, "yaml", gpath.DefaultGlideFile, "Set a YAML configuration file")
	flag.BoolVar(&argDebug, "debug", false, "Print debug verbose informational messages")
	flag.BoolVar(&argQuiet, "quiet", false, "Quiet (no info or debug messages)")
}

func main() {
	flag.Parse()

	action.Debug(argDebug)
	action.Quiet(argQuiet)
	gpath.GlideFile = glideYaml

	// load package from glide.yml config
	msg.Debug("Loading Glide config from %s...", glideYaml)
	glideConfig := loadGlideConfig()

	msg.Debug("Collecting imported packages...")
	importPkgs := make(map[string]interface{})
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
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

		msg.Debug("--> getting import package in %q", path)
		pkgs, err := getImports(path)
		if err == nil {
			for _, pkg := range pkgs {
				importPkgs[pkg] = nil
			}
		} else {
			msg.Err("Error when get import package for file %v: %v", path, err)
		}

		return nil
	})

	if err != nil {
		msg.Die(err.Error())
	}

	msg.Debug("Checking unused packages...")
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

		msg.Debug("--> package %q is not used", dep.Name)
		unusedPkgs[dep.Name] = nil
	}

	if len(unusedPkgs) == 0 {
		msg.Info("Well done! All packages are needed.")
		os.Exit(0)
	}

	msg.Debug("Removing unused packages...")
	deps := make([]*cfg.Dependency, 0, len(glideConfig.Imports))
	for _, pkg := range glideConfig.Imports {
		if _, unused := unusedPkgs[pkg.Name]; !unused {
			deps = append(deps, pkg)
		}
	}
	glideConfig.Imports = deps

	glideYamlFile, _ := gpath.Glide()
	if err = glideConfig.WriteFile(glideYamlFile); err != nil {
		msg.Die("Error while write Glide config to file back: %v", err)
	}

	msg.Info("New Glide config has been updated with removing unused packages.")
}

func loadGlideConfig() (config *cfg.Config) {
	glideYamlFile, err := gpath.Glide()
	if err != nil {
		msg.Die("Could not find Glide config file")
	}

	var yml []byte
	if yml, err = ioutil.ReadFile(glideYamlFile); err != nil {
		msg.Die("Error while reading config file: %v", err)
	}

	if config, err = cfg.ConfigFromYaml(yml); err != nil {
		msg.Die("Error while parsing config file: %v", err)
	}

	return config
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
