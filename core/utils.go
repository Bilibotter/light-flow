package core

import (
	"reflect"
	"runtime"
	"strings"
)

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
		return splits[len(splits)-1]
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
