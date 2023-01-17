package pkg

import (
	"os"
	"strings"

	"golang.org/x/mod/modfile"
)

type Module struct {
	Name    string
	Require map[string]string
	Replace map[string]string
}

func ReadMod(path string) (Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Module{}, err
	}
	file, err := modfile.Parse(path, data, nil)
	if err != nil {
		return Module{}, err
	}
	mod := Module{
		Name: file.Module.Mod.Path,
	}
	mod.Require = make(map[string]string, len(file.Require))
	for _, require := range file.Require {
		mod.Require[require.Mod.Path] = require.Mod.Version
	}
	mod.Replace = make(map[string]string, len(file.Replace))
	for _, replace := range file.Replace {
		mod.Replace[replace.Old.Path] = replace.New.Path
	}
	return mod, nil
}

func (m Module) SplitModulePackage(url string) (string, string) {
	for pkg := range m.Require {
		if strings.HasPrefix(url, pkg) {
			return url[:len(pkg)], url[len(pkg)+1:]
		}
	}
	return "", ""
}
