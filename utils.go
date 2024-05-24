package light_flow

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

type set[T comparable] struct {
	data map[T]bool
}

func newRoutineUnsafeSet[T comparable]() *set[T] {
	s := &set[T]{}
	s.data = make(map[T]bool)
	return s
}

func createSetBySliceFunc[T any, K comparable](src []T, transfer func(T) K) *set[K] {
	result := newRoutineUnsafeSet[K]()
	for _, ele := range src {
		result.Add(transfer(ele))
	}
	return result
}

func (s *set[T]) Slice() []T {
	result := make([]T, 0, len(s.data))
	for key := range s.data {
		result = append(result, key)
	}
	return result
}

func (s *set[T]) Add(item T) {
	s.data[item] = true
}

func (s *set[T]) Contains(item T) bool {
	return s.data[item]
}

func (s *set[T]) Remove(item T) {
	delete(s.data, item)
}

func (s *set[T]) Size() int {
	return len(s.data)
}

func getStructName(obj any) string {
	if reflect.ValueOf(obj).Kind() == reflect.Ptr {
		return "*" + reflect.TypeOf(obj).Elem().Name()
	}
	return reflect.TypeOf(obj).Name()
}

// getFuncName function retrieves the stepName of a provided function.
// If the provided function is anonymous function, it panics.
func getFuncName(f interface{}) string {
	funcValue := reflect.ValueOf(f)
	funcType := funcValue.Type()

	if funcType.Kind() != reflect.Func {
		panic("param must be func")
	}

	absoluteName := runtime.FuncForPC(funcValue.Pointer()).Name()
	if len(absoluteName) == 0 {
		panic("It is not allowed to use getFuncName to an anonymous function.")
	}
	if strings.Contains(absoluteName, ".") {
		splits := strings.Split(absoluteName, ".")
		absoluteName = splits[len(splits)-1]
	}

	pattern := `^func\d+`
	match, _ := regexp.MatchString(pattern, absoluteName)
	if match {
		panic(fmt.Sprintf("FuncName[%s] is like anonymous function name", absoluteName))
	}
	return absoluteName
}

// copyPropertiesSkipNotEmpty will skip field contain`skip` tag value
func copyPropertiesSkipNotEmpty(src, dst interface{}) {
	copyProperties(src, dst, true)
}

func copyPropertiesWithMerge(src, dst interface{}) {
	copyProperties(src, dst, false)
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
		// private field can't copy, skip it
		if len(srcType.Field(i).PkgPath) != 0 {
			continue
		}

		if tag := srcType.Field(i).Tag.Get("flow"); len(tag) != 0 {
			splits := strings.Split(tag, ";")
			skip := false
			for j := 0; j < len(splits) && !skip; j++ {
				skip = splits[j] == "skip"
			}
			if skip {
				continue
			}
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

func createStruct[T any](src any) (target T) {
	srcValue := reflect.ValueOf(src)
	if srcValue.Kind() != reflect.Ptr {
		panic("src must be a pointer")
	}

	if reflect.TypeOf(target).Kind() != reflect.Struct {
		panic("The generic type is not a struct")
	}

	copyPropertiesWithMerge(src, &target)

	return target
}
