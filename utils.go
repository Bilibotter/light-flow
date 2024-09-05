package light_flow

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"unsafe"
)

var (
	dunno       = "???"
	centerDot   = "·"
	dot         = "."
	slash       = "/"
	pattern     = `^[a-zA-Z0-9]+$`
	patternHint = "the name must consist of a combination of uppercase and lowercase English letters and numbers"
)

type empty struct{}

type set[T comparable] struct {
	Data map[T]empty
}

func isValidIdentifier(identifier string) bool {
	// 正则表达式匹配大小写英文字母及数字
	var validIdentifier = regexp.MustCompile(pattern)
	return validIdentifier.MatchString(identifier)
}

// This file includes a modified version of the 'stack' function from the 'gin' framework,
// originally licensed under the MIT License at https://github.com/gin-gonic/gin.
func stack() []byte {
	limitSize := 8182
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := 3; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Skip assemble file
		if strings.HasSuffix(file, ".s") {
			continue
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		funcName := function(pc)
		if funcName == dunno {
			continue // Skip printing if function name is "???" or starts with "runtime."
		}
		fmt.Fprintf(buf, "\t%s -> %s\n", funcName, source(lines, line))
		if buf.Len() > limitSize {
			break
		}
	}
	return buf.Bytes()
}

func function(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := fn.Name()
	// Benchmark tests have shown that using 'strings' package outperforms 'bytes' for this particular use case.
	if lastSlash := strings.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := strings.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = strings.ReplaceAll(name, centerDot, dot)
	return name
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return []byte(dunno)
	}
	return bytes.TrimSpace(lines[n])
}

func newRoutineUnsafeSet[T comparable](elem ...T) *set[T] {
	s := &set[T]{}
	s.Data = make(map[T]empty, len(elem))
	for _, e := range elem {
		s.Add(e)
	}
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
	result := make([]T, 0, len(s.Data))
	for key := range s.Data {
		result = append(result, key)
	}
	return result
}

func (s *set[T]) Add(item T) {
	s.Data[item] = empty{}
}

func (s *set[T]) Contains(item T) (exists bool) {
	_, exists = s.Data[item]
	return
}

func (s *set[T]) Remove(item T) {
	delete(s.Data, item)
}

func (s *set[T]) Size() int {
	return len(s.Data)
}

func getStructName(obj any) string {
	if reflect.ValueOf(obj).Kind() == reflect.Ptr {
		return "*" + reflect.TypeOf(obj).Elem().Name()
	}
	return reflect.TypeOf(obj).Name()
}

// getFuncName function retrieves the name of a provided function.
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

	p := `^func\d+`
	match, _ := regexp.MatchString(p, absoluteName)
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
		srcFieldName := srcType.Field(i).Name
		if !srcField.CanInterface() {
			srcField = reflect.NewAt(srcField.Type(), unsafe.Pointer(srcField.UnsafeAddr())).Elem()
		}
		if dstField := dstElem.FieldByName(srcFieldName); dstField.IsValid() && dstField.Type() == srcField.Type() {
			if !dstField.CanSet() {
				dstField = reflect.NewAt(dstField.Type(), unsafe.Pointer(dstField.UnsafeAddr())).Elem()
			}
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
