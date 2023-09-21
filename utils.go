package light_flow

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

type Set struct {
	data map[string]bool
}

func NewRoutineUnsafeSet() *Set {
	s := &Set{}
	s.data = make(map[string]bool)
	return s
}

func (s *Set) Add(item string) {
	s.data[item] = true
}

func (s *Set) Contains(item string) bool {
	return s.data[item]
}

func (s *Set) Remove(item string) {
	delete(s.data, item)
}

func (s *Set) Size() int {
	return len(s.data)
}

func GetStructName(obj any) string {
	if reflect.ValueOf(obj).Kind() == reflect.Ptr {
		return "*" + reflect.TypeOf(obj).Elem().Name()
	}
	return reflect.TypeOf(obj).Name()
}

// GetFuncName function retrieves the stepName of a provided function.
// If the provided function is anonymous function, it panics.
func GetFuncName(f interface{}) string {
	funcValue := reflect.ValueOf(f)
	funcType := funcValue.Type()

	if funcType.Kind() != reflect.Func {
		panic("param must be func")
	}

	absoluteName := runtime.FuncForPC(funcValue.Pointer()).Name()
	if len(absoluteName) == 0 {
		panic("It is not allowed to use GetFuncName to an anonymous function.")
	}
	if strings.Contains(absoluteName, ".") {
		splits := strings.Split(absoluteName, ".")
		absoluteName = splits[len(splits)-1]
	}

	pattern := `^func\d+`
	match, _ := regexp.MatchString(pattern, absoluteName)
	if match {
		panic(fmt.Sprintf("func name %s is like anonymous function name", absoluteName))
	}
	return absoluteName
}

func CopyProperties(src, dst interface{}) {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	if srcValue.Kind() != reflect.Ptr || dstValue.Kind() != reflect.Ptr {
		panic("Both src and dst must be pointers")
	}

	srcElem := srcValue.Elem()
	dstElem := dstValue.Elem()
	srcType := srcElem.Type()
	for i := 0; i < srcElem.NumField(); i++ {
		srcField := srcElem.Field(i)
		srcFieldName := srcType.Field(i).Name

		if dstField := dstElem.FieldByName(srcFieldName); dstField.IsValid() && dstField.Type() == srcField.Type() {
			if !srcField.IsZero() {
				dstField.Set(srcField)
			}
		}
	}
}

func CreateStruct[T any](src any) (target T) {
	srcValue := reflect.ValueOf(src)
	if srcValue.Kind() != reflect.Ptr {
		panic("src must be a pointer")
	}

	if reflect.TypeOf(target).Kind() != reflect.Struct {
		panic("The generic type is not a struct")
	}

	CopyProperties(src, &target)

	return target
}
