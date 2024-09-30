package codegen

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/samber/lo"

	"github.com/arklib/ark/util"
)

type Type struct {
	pkg      *Package
	rType    reflect.Type
	isGen    bool
	isRename bool
	code     string

	Key      string
	Mod      string
	PkgPath  string
	PkgPaths []string
	PkgName  string
	RefName  string
	Name     string
	Prefix   string
	FlatName string
}

func newType(pkg *Package, rType reflect.Type, prefix string) *Type {
	t := &Type{
		pkg:     pkg,
		rType:   rType,
		PkgPath: rType.PkgPath(),
		Name:    rType.Name(),
		Prefix:  prefix,
	}

	if t.PkgPath != "" {
		paths, name := util.SplitSuffix(t.PkgPath, "/")
		t.Mod = paths[0]
		t.PkgPaths = paths
		t.PkgName = name
		t.Key = fmt.Sprintf("%s.%s", t.PkgPath, t.Name)
		t.RefName = fmt.Sprintf("%s.%s", t.PkgName, t.Name)

		switch {
		case t.PkgName == pkg.name:
			t.FlatName = lo.PascalCase(t.Prefix) + t.Name
			t.isRename = true
		case t.Mod == pkg.mod:
			t.FlatName = fmt.Sprintf("%s_%s", strings.ToUpper(t.PkgName), t.Name)
			t.isRename = true
		default:
			t.isGen = true
			t.FlatName = t.RefName
			t.pkg.AddImport(t.PkgPath)
		}
	}
	return t
}

func (t *Type) GenCode() {
	if t.isGen {
		return
	}
	t.isGen = true
	rType := t.rType

	depTypes := make([]reflect.Type, 0)
	t.write("type %s ", t.FlatName)

	switch rType.Kind() {
	case reflect.Struct:
		t.writeln("struct {")
		for i := 0; i < rType.NumField(); i++ {
			field := rType.Field(i)
			if !field.IsExported() {
				continue
			}

			// field code like (name  type  `json:"name"`)
			t.write("    %s ", field.Name)
			fieldType := t.refCode(field.Type)

			tag := t.buildTag(fieldType, field.Tag)
			t.writeln(" `%s`", tag)

			depTypes = append(depTypes, fieldType)
		}
		t.writeln("}")
	default:
		depTypes = append(depTypes, t.refCodeX(rType))
	}

	for _, dType := range depTypes {
		t.pkg.loadType(dType)
	}
}

func (t *Type) buildTag(field reflect.Type, tag reflect.StructTag) (tagStr string) {
	tagStr = string(tag)

	tInfo := t.pkg.loadType(field)
	if tInfo == nil || !tInfo.isRename {
		return
	}

	frugal, ok := tag.Lookup("frugal")
	if ok {
		frugalVal := strings.ReplaceAll(frugal, tInfo.Name, tInfo.FlatName)
		oldFrugal := fmt.Sprintf(`frugal:"%s"`, frugal)
		newFrugal := fmt.Sprintf(`frugal:"%s"`, frugalVal)
		tagStr = strings.ReplaceAll(tagStr, oldFrugal, newFrugal)
	}
	return
}

func (t *Type) refCode(rType reflect.Type) reflect.Type {
	if rType.PkgPath() != "" {
		t.write(t.pkg.loadType(rType).FlatName)
		return rType
	}
	return t.refCodeX(rType)
}

func (t *Type) refCodeX(rType reflect.Type) reflect.Type {
	switch rType.Kind() {
	case reflect.Ptr:
		t.write("*").refCode(rType.Elem())
	case reflect.Slice:
		t.write("[]").refCode(rType.Elem())
	case reflect.Map:
		t.write("map[")
		t.refCode(rType.Key())
		t.write("]")
		t.refCode(rType.Elem())
	case reflect.Interface:
		t.write("any")
	case reflect.Struct:
		t.write("%s", rType.Name())
	default:
		t.write("%s", rType.Name())
	}
	return rType
}

// Source code
func (t *Type) Source() string {
	return t.code
}

func (t *Type) write(code string, a ...any) *Type {
	t.code += fmt.Sprintf(code, a...)
	return t
}

func (t *Type) writeln(code string, a ...any) *Type {
	return t.write(code+"\n", a...)
}
