// Package csv provides functionality for decoding rows of a CSV file into a struct.
package csv

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ErrUseDefault can be used by custom assign functions for indicating that the unmarshaller should fallback
// to the default assign function for the given type.
var ErrUseDefault = errors.New("use default assign function")

// Reader is the interface used by the decoder to read CSV input line by line.
//
// Read returns the next line of values, split into a slice of strings
type Reader interface {
	Read() ([]string, error)
}

// AssignFn is the signature for custom assign functions
type AssignFn func(s string, v reflect.Value, tag reflect.StructTag) error

// A Decoder reads input from a Reader and parses the values into a struct.
//
// The decoder supports the following struct field types by default:
// int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, string,
// time.Time (requires custom attribute -- see example). Empty string values returned by the internal reader
// are treated as the nil value of the given type (ie 0 for numeric types). Additional types, or modification
// of the default behaviour, can be overridden by setting custom assign functions.
type Decoder struct {
	Indexes   map[string]int
	assigners map[reflect.Kind]AssignFn
	r         Reader
	line      int
}

// NewDecoder returns a new Decoder instance using r as its source
func NewDecoder(r Reader) *Decoder {
	return &Decoder{
		r:         r,
		assigners: getDefaultAssigners(),
	}
}

// SetAssignFn sets the given function as the assign function for fields of the given kind, overriding any
// previous behaviour for that kind.
func (this *Decoder) SetAssignFn(kind reflect.Kind, fn AssignFn) {
	this.assigners[kind] = fn
}

// ReadHeader reads the current line of the decoder's Reader and interprets it as a header with field names,
// setting the decoder's Indexes field accordingly.
//
// ReaderHeader must be called only once, and before any calls to Decode()
func (this *Decoder) ReadHeader() error {
	if this.line > 0 {
		return errors.New("ReadHeader can only be called once, and only before Decode()")
	}
	this.line += 1
	if data, err := this.r.Read(); err != nil {
		return err
	} else {
		this.Indexes = make(map[string]int)
		for i, d := range data {
			this.Indexes[d] = i
		}
	}
	return nil
}

// Decode reads the next values from its reader, parses the data and stores the result in the value
// pointed to by dst. If the internal reader returns an error, that error is returned.
//
// There are multiple ways in which the values read can be decoded into the struct. See the examples for
// the possible options.
func (this *Decoder) Decode(dst interface{}) error {
	this.line += 1
	if data, err := this.r.Read(); err != nil {
		return err
	} else {
		if err = unmarshall(data, this.Indexes, this.assigners, dst); err != nil && err != io.EOF {
			return fmt.Errorf("csv: Error on line %d: %v", this.line, err)
		}
		return nil
	}
}

func unmarshall(data []string, indexes map[string]int, assigners map[reflect.Kind]AssignFn, dst interface{}) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr {
		return errors.New("csv: dst is not a pointer")
	}
	if val.IsNil() {
		return errors.New("csv: dst is nil")
	}
	e := val.Elem()
	n := e.NumField()

	// If indexes is not specified, field number i of dst gets assigned to data[i].
	// If the number of fields in dst and number of rows in data is inequal, we treat it as an error
	if indexes == nil && n != len(data) {
		return errors.New("csv: struct field count didn't match data column count")
	}

	t := e.Type()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		var dataIndex int
		if indexes != nil {
			if dataIndex = fieldIndex(f, indexes); dataIndex == -1 {
				continue
			}
		} else {
			dataIndex = i
		}

		s := data[dataIndex]
		v := e.Field(i)

		if a, ok := assigners[v.Kind()]; ok {
			if err := a(s, v, f.Tag); err != nil {
				if err == ErrUseDefault {
					if a, ok := defaultAssigners[v.Kind()]; ok {
						err = a(s, v, f.Tag)
					}
				}
				if err != nil {
					return fmt.Errorf("csv: error assigning value to field %s: %v", f.Name, err)
				}
			}
		} else {
			return fmt.Errorf("csv: unassignable field type for field %s: %v", f.Name, v.Kind())
		}
	}
	return nil
}

var defaultAssigners = getDefaultAssigners()

func getDefaultAssigners() map[reflect.Kind]AssignFn {
	dict := make(map[reflect.Kind]AssignFn)
	dict[reflect.String] = assignString
	dict[reflect.Struct] = assignStruct
	for _, kind := range []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64} {
		dict[kind] = assignInt
	}
	for _, kind := range []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr} {
		dict[kind] = assignUint
	}
	for _, kind := range []reflect.Kind{reflect.Float32, reflect.Float64} {
		dict[kind] = assignFloat
	}
	return dict
}

func assignString(s string, v reflect.Value, tag reflect.StructTag) error {
	v.SetString(s)
	return nil
}

func assignInt(s string, v reflect.Value, tag reflect.StructTag) error {
	if s == "" {
		v.SetInt(0)
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err != nil {
		return err
	} else if v.OverflowInt(n) {
		return fmt.Errorf("overflow")
	} else {
		v.SetInt(n)
		return nil
	}
}

func assignUint(s string, v reflect.Value, tag reflect.StructTag) error {
	if s == "" {
		v.SetUint(0)
		return nil
	}
	if n, err := strconv.ParseUint(s, 10, 64); err != nil {
		return err
	} else if v.OverflowUint(n) {
		return fmt.Errorf("overflow")
	} else {
		v.SetUint(n)
		return nil
	}
}

func assignFloat(s string, v reflect.Value, tag reflect.StructTag) error {
	if s == "" {
		v.SetFloat(0.0)
		return nil
	}
	if n, err := strconv.ParseFloat(s, v.Type().Bits()); err != nil {
		return err
	} else if v.OverflowFloat(n) {
		return fmt.Errorf("overflow")
	} else {
		v.SetFloat(n)
		return nil
	}
}

func assignStruct(s string, v reflect.Value, tag reflect.StructTag) error {
	switch v.Interface().(type) {
	case time.Time:
		if s == "" {
			v.Set(reflect.ValueOf(time.Time{}))
			return nil
		}
		if split := strings.Split(tag.Get("csv"), ","); len(split) > 1 {
			format := split[1]
			if t, err := time.Parse(format, s); err != nil {
				return err
			} else {
				v.Set(reflect.ValueOf(t))
				return nil
			}
		} else {
			return fmt.Errorf("missing format info in tag")
		}
	default:
		return fmt.Errorf("unsupported struct type: %s", v.Kind())
	}
}

func fieldIndex(f reflect.StructField, indexes map[string]int) int {
	var name string
	if tag := f.Tag.Get("csv"); tag != "" {
		name = strings.Split(tag, ",")[0]
	} else {
		name = f.Name
	}
	if i, ok := indexes[name]; ok {
		return i
	} else {
		return -1
	}
}
