package figo

import (
	"reflect"
	"strconv"
)

// StructToMapString converts struct as map string
func StructToMapString(i interface{}) map[string][]string {
	ms := map[string][]string{}
	iv := reflect.ValueOf(i).Elem()
	tp := iv.Type()

	for i := 0; i < iv.NumField(); i++ {
		k := tp.Field(i).Name
		f := iv.Field(i)
		ms[k] = ValueToString(f)
	}

	return ms
}

// ValueToString converts supported type of f as slice string
func ValueToString(f reflect.Value) []string {
	var v []string

	switch reflect.TypeOf(f.Interface()).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v = []string{strconv.FormatInt(f.Int(), 10)}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v = []string{strconv.FormatUint(f.Uint(), 10)}
	case reflect.Float32:
		v = []string{strconv.FormatFloat(f.Float(), 'f', 4, 32)}
	case reflect.Float64:
		v = []string{strconv.FormatFloat(f.Float(), 'f', 4, 64)}
	case reflect.Bool:
		v = []string{strconv.FormatBool(f.Bool())}
	case reflect.Slice:
		for i := 0; i < f.Len(); i++ {
			if s := ValueToString(f.Index(i)); len(s) == 1 {
				v = append(v, s[0])
			}
		}
	case reflect.String:
		v = []string{f.String()}
	}

	return v
}
