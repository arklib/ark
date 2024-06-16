package codegen

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/arklib/ark/util"
)

type Types map[string]*Type

type Package struct {
	mod     string
	name    string
	outName string
	imports map[string]bool
	types   Types
	structs Types
}

func NewPackage(name string) *Package {
	return &Package{
		outName: name,
		imports: make(map[string]bool),
		types:   make(Types),
		structs: make(Types),
	}
}

func (p *Package) loadType(rType reflect.Type) (t *Type) {
	if rType.PkgPath() == "" {
		return
	}

	id := fmt.Sprintf("%s.%s", rType.PkgPath(), rType.Name())
	t, ok := p.types[id]
	if !ok {
		t = newType(p, rType)
		p.types[id] = t
		t.GenCode()
	}
	return
}

func (p *Package) AddImport(name string) {
	p.imports[name] = true
}

func (p *Package) GetTypes() Types {
	return p.types
}

func (p *Package) AddStruct(val any) (t *Type) {
	rType := reflect.TypeOf(val)
	for {
		if rType.Kind() != reflect.Ptr {
			break
		}
		rType = rType.Elem()
	}

	pkgPath := rType.PkgPath()

	if p.mod == "" {
		paths, name := ParsePkgPath(pkgPath)
		p.mod = paths[0]
		p.name = name
	}

	t = p.loadType(rType)
	p.structs[t.Key] = t
	return
}

func (p *Package) Source() string {
	for _, t := range p.structs {
		t.GenCode()
	}

	codes := []string{
		fmt.Sprintf("package %s\n", p.outName),
	}

	util.ForEachMapBySort(p.imports, func(key string, _ bool) {
		code := fmt.Sprintf(`import "%s"`, key)
		codes = append(codes, code)
	})

	util.ForEachMapBySort(p.types, func(_ string, t *Type) {
		codes = append(codes, t.Source())
	})
	return strings.Join(codes, "\n")
}

func (p *Package) Dump() {
	fmt.Println(p.Source())
}

func ParsePkgPath(path string) ([]string, string) {
	paths := strings.Split(path, "/")
	name := paths[len(paths)-1]
	return paths, name
}
