package env

import (
	"encoding"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type CustomParsers map[reflect.Type]ParserFunc
type ParserFunc func(v string) (interface{}, error)

func Parse(v interface{}) error {
	return ParseWithFuncs(v, CustomParsers{})
}

func ParseWithFuncs(v interface{}, funcMap CustomParsers) error {

	ptrRef := reflect.ValueOf(v)

	if ptrRef.Kind() != reflect.Ptr {
		return ErrNotAStructPtr
	}

	ref := ptrRef.Elem()

	if ref.Kind() != reflect.Struct {
		return ErrNotAStructPtr
	}
	return doParse(ref, funcMap)
}

func doParse(ref reflect.Value, funcMap CustomParsers) (err error) {

	refType := ref.Type()

	for i := 0; i < refType.NumField(); i++ {

		refField := ref.Field(i)

		if reflect.Ptr == refField.Kind() && !refField.IsNil() && refField.CanSet() {
			if err = Parse(refField.Interface()); err != nil {
				return
			}
			continue
		}

		var value string
		refTypeField := refType.Field(i)

		if value, err = get(refTypeField); err != nil {
			return
		}

		if value == "" {
			if reflect.Struct == refField.Kind() {
				if err := doParse(refField, funcMap); err != nil {
					return err
				}
			}
			continue
		}

		if err = set(refField, refTypeField, value, funcMap); err != nil {
			return err
		}
	}
	return
}

func get(field reflect.StructField) (val string, err error) {

	key, opts := parseKeyForOption(field.Tag.Get("env"))

	defaultValue := field.Tag.Get("envDefault")
	val = getOr(key, defaultValue)

	expandVar := field.Tag.Get("envExpand")

	if strings.ToLower(expandVar) == "true" {
		val = os.ExpandEnv(val)
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			switch opt {
			case "":
				break
			case "required":
				val, err = getRequired(key)
			default:
				err = fmt.Errorf("env: tag option %q not supported", opt)
			}
		}
	}
	return
}

func parseKeyForOption(key string) (string, []string) {

	opts := strings.Split(key, ",")
	return opts[0], opts[1:]
}

func getRequired(key string) (value string, err error) {

	var ok bool
	if value, ok = os.LookupEnv(key); ok {
		return
	}
	err = fmt.Errorf(`env: required environment variable "%q" is not set`, key)
	return
}

func getOr(key, defaultValue string) string {

	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}

func set(field reflect.Value, sf reflect.StructField, value string, funcMap CustomParsers) (err error) {

	if field.Kind() == reflect.Slice {
		return handleSlice(field, value, sf, funcMap)
	}

	var ok bool
	var val interface{}
	var parserFunc ParserFunc

	if parserFunc, ok = funcMap[sf.Type]; ok {
		if val, err = parserFunc(value); err != nil {
			return newParseError(sf, err)
		}
		field.Set(reflect.ValueOf(val))
		return
	}

	if parserFunc, ok = defaultBuiltInParsers[sf.Type.Kind()]; ok {

		if val, err = parserFunc(value); err != nil {
			return newParseError(sf, err)
		}

		field.Set(reflect.ValueOf(val).Convert(sf.Type))
		return
	}

	return handleTextUnmarshaller(field, value, sf)
}

func handleSlice(field reflect.Value, value string, sf reflect.StructField, funcMap CustomParsers) error {

	var separator = sf.Tag.Get("envSeparator")
	if separator == "" {
		separator = ","
	}
	var parts = strings.Split(value, separator)

	var elemType = sf.Type.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if _, ok := reflect.New(elemType).Interface().(encoding.TextUnmarshaler); ok {
		return parseTextUnmarshallers(field, parts, sf)
	}

	parserFunc, ok := funcMap[elemType]

	if !ok {
		parserFunc, ok = defaultBuiltInParsers[elemType.Kind()]
		if !ok {
			return newNoParserError(sf)
		}
	}

	var result = reflect.MakeSlice(sf.Type, 0, len(parts))

	for _, part := range parts {

		r, err := parserFunc(part)
		if err != nil {
			return newParseError(sf, err)
		}
		var v = reflect.ValueOf(r).Convert(elemType)
		if sf.Type.Elem().Kind() == reflect.Ptr {
			// TODO: add this!
			return fmt.Errorf("env: point slices of built-in and aliased types are not supported: %s %s", sf.Name, sf.Type)
		}
		result = reflect.Append(result, v)
	}

	field.Set(result)
	return nil
}

func handleTextUnmarshaller(field reflect.Value, value string, sf reflect.StructField) error {

	if reflect.Ptr == field.Kind() {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
	} else if field.CanAddr() {
		field = field.Addr()
	}

	tm, ok := field.Interface().(encoding.TextUnmarshaler)

	if !ok {
		return newNoParserError(sf)
	}

	var err = tm.UnmarshalText([]byte(value))
	return newParseError(sf, err)
}

func parseTextUnmarshallers(field reflect.Value, data []string, sf reflect.StructField) error {

	s := len(data)
	elemType := field.Type().Elem()
	slice := reflect.MakeSlice(reflect.SliceOf(elemType), s, s)

	for i, v := range data {
		sv := slice.Index(i)
		kind := sv.Kind()
		if kind == reflect.Ptr {
			sv = reflect.New(elemType.Elem())
		} else {
			sv = sv.Addr()
		}
		tm := sv.Interface().(encoding.TextUnmarshaler)
		if err := tm.UnmarshalText([]byte(v)); err != nil {
			return newParseError(sf, err)
		}
		if kind == reflect.Ptr {
			slice.Index(i).Set(sv)
		}
	}
	field.Set(slice)
	return nil
}

func newParseError(sf reflect.StructField, err error) error {

	if err == nil {
		return nil
	}
	return parseError{
		sf:  sf,
		err: err,
	}
}

type parseError struct {
	err error
	sf  reflect.StructField
}

func (e parseError) Error() string {
	return fmt.Sprintf(`env: parse error on field "%s" of type "%s": %v`, e.sf.Name, e.sf.Type, e.err)
}

func newNoParserError(sf reflect.StructField) error {
	return fmt.Errorf(`env: no parser found for field "%s" of type "%s"`, sf.Name, sf.Type)
}

var (
	ErrNotAStructPtr = errors.New("env: expected a pointer to a Struct")

	defaultBuiltInParsers = map[reflect.Kind]ParserFunc{

		reflect.Bool: func(v string) (interface{}, error) {
			return strconv.ParseBool(v)
		},

		reflect.String: func(v string) (interface{}, error) {
			return v, nil
		},

		reflect.Int: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 32)
			return int(i), err
		},

		reflect.Int16: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 16)
			return int16(i), err
		},

		reflect.Int32: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 32)
			return int32(i), err
		},

		reflect.Int64: func(v string) (interface{}, error) {
			return strconv.ParseInt(v, 10, 64)
		},

		reflect.Int8: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 8)
			return int8(i), err
		},

		reflect.Uint: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 32)
			return uint(i), err
		},

		reflect.Uint16: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 16)
			return uint16(i), err
		},

		reflect.Uint32: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 32)
			return uint32(i), err
		},

		reflect.Uint64: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 64)
			return i, err
		},

		reflect.Uint8: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 8)
			return uint8(i), err
		},

		reflect.Float64: func(v string) (interface{}, error) {
			return strconv.ParseFloat(v, 64)
		},

		reflect.Float32: func(v string) (interface{}, error) {
			f, err := strconv.ParseFloat(v, 32)
			return float32(f), err
		},
	}
)
