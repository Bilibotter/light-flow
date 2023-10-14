package light_flow

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

type Set[T comparable] struct {
	data map[T]bool
}

func NewRoutineUnsafeSet[T comparable]() *Set[T] {
	s := &Set[T]{}
	s.data = make(map[T]bool)
	return s
}

func CreateFromSliceFunc[T any, K comparable](src []T, transfer func(T) K) *Set[K] {
	result := NewRoutineUnsafeSet[K]()
	for _, ele := range src {
		result.Add(transfer(ele))
	}
	return result
}

func (s *Set[T]) Slice() []T {
	result := make([]T, 0, len(s.data))
	for key := range s.data {
		result = append(result, key)
	}
	return result
}

func (s *Set[T]) Add(item T) {
	s.data[item] = true
}

func (s *Set[T]) Contains(item T) bool {
	return s.data[item]
}

func (s *Set[T]) Remove(item T) {
	delete(s.data, item)
}

func (s *Set[T]) Size() int {
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

func CopyPropertiesSkipNotEmpty(src, dst interface{}) {
	copyProperties(src, dst, true)
}

func CopyProperties(src, dst interface{}) {
	copyProperties(src, dst, false)
	//srcValue := reflect.ValueOf(src)
	//dstValue := reflect.ValueOf(dst)
	//
	//if srcValue.Kind() != reflect.Ptr || dstValue.Kind() != reflect.Ptr {
	//	panic("Both src and dst must be pointers")
	//}
	//
	//srcElem := srcValue.Elem()
	//dstElem := dstValue.Elem()
	//srcType := srcElem.Type()
	//for i := 0; i < srcElem.NumField(); i++ {
	//	srcField := srcElem.Field(i)
	//	srcFieldName := srcType.Field(i).GetCtxName
	//
	//	if dstField := dstElem.FieldByName(srcFieldName); dstField.IsValid() && dstField.Type() == srcField.Type() {
	//		if !srcField.IsZero() {
	//			dstField.Set(srcField)
	//		}
	//	}
	//}
}

func copyProperties(src, dst interface{}, skipNotEmpty bool) {
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

		if len(srcType.Field(i).PkgPath) != 0 {
			continue
		}

		if dstField := dstElem.FieldByName(srcFieldName); dstField.IsValid() && dstField.Type() == srcField.Type() {
			if skipNotEmpty && !dstField.IsZero() {
				continue
			}
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
