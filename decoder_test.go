package csvdecoder

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestUnmarshall(t *testing.T) {
	type thing struct {
		S string
		I int64
		F float64
	}
	type taggedThing struct {
		S string  `csv:"myString"`
		I int64   `csv:"int"`
		F float64 `csv:"float-prop"`
	}
	type timeThing struct {
		D time.Time
	}
	type taggedTimeThing struct {
		D time.Time `csv:",2006-01-02"`
	}
	type nameAndFormatTaggedTimeThing struct {
		D time.Time `csv:"date,2006-01-02"`
	}
	var nilDst *thing
	tests := []struct {
		name string
		data []string
		idx  map[string]int
		ass  map[reflect.Kind]AssignFn
		dst  interface{}
		want interface{}
		err  error
	}{
		{
			name: "Error if dst is not pointer",
			data: nil,
			dst:  thing{},
			want: thing{},
			err:  fmt.Errorf("csvdecoder: dst is not a pointer"),
		},
		{
			name: "Error if dst is nil",
			data: nil,
			dst:  nilDst,
			want: nilDst,
			err:  fmt.Errorf("csvdecoder: dst is nil"),
		},
		{
			name: "error if no indexes and len(struct fields) != len(data)",
			data: []string{"str"},
			idx:  nil,
			dst:  &thing{},
			want: &thing{},
			err:  fmt.Errorf("struct field count didn't match data column count"),
		},
		{
			name: "if no indexes, load fields according their index index in the struct",
			data: []string{"str", "1", "1.5"},
			idx:  nil,
			dst:  &thing{},
			want: &thing{"str", 1, 1.5},
		},
		{
			name: "if indexes and no tag, load according to field name's index",
			data: []string{"1.5", "str", "1"},
			idx:  map[string]int{"F": 0, "S": 1, "I": 2},
			dst:  &thing{},
			want: &thing{"str", 1, 1.5},
		},
		{
			name: "if indexes and tag, load according to field tag's index",
			data: []string{"str", "1.5", "1"}, // the I and F fields are switched around, so the "no indexes" strategy won't work
			idx:  map[string]int{"myString": 0, "float-prop": 1, "int": 2},
			dst:  &taggedThing{},
			want: &taggedThing{"str", 1, 1.5},
		},
		{
			name: "ignores fields not present in indexes",
			data: []string{"1.5", "str", "1"},
			idx:  map[string]int{"F": 0 /*{"S": 1},*/, "I": 2}, // don't specify field S's index. S should be ignored
			dst:  &thing{},
			want: &thing{"", 1, 1.5},
		},
		{
			name: "error if no assigner for field kind",
			data: []string{"str", "1", "1.5"},
			idx:  nil,
			ass: map[reflect.Kind]AssignFn{ // support Int64 and Float64, but not string
				reflect.Int64:   defaultAssigners[reflect.Int64],
				reflect.Float64: defaultAssigners[reflect.Float64],
			},
			dst:  &thing{},
			want: &thing{},
			err:  fmt.Errorf("csvdecoder: unassignable field type for field S: string"),
		},
		{
			name: "returns assigner error if assigner returns error",
			data: []string{"str", "1", "1.5"},
			idx:  nil,
			ass: map[reflect.Kind]AssignFn{ // support Int64 and Float64, but not string
				reflect.Int64:   defaultAssigners[reflect.Int64],
				reflect.Float64: defaultAssigners[reflect.Float64],
				reflect.String:  func(s string, v reflect.Value, t reflect.StructTag) error { return fmt.Errorf("err!") },
			},
			dst:  &thing{},
			want: &thing{},
			err:  fmt.Errorf("csvdecoder: error assigning value to field S: err!"),
		},
		{
			name: "error if field type is time and tag doesn't contain format",
			data: []string{"1988-11-08"},
			dst:  &timeThing{},
			want: &timeThing{},
			err:  fmt.Errorf("csvdecoder: error assigning value to field D: missing format info in tag"),
		},
		{
			name: "parses time according to format in tag",
			data: []string{"1988-11-08"},
			idx:  nil,
			dst:  &taggedTimeThing{},
			want: &taggedTimeThing{time.Date(1988, time.November, 8, 0, 0, 0, 0, time.UTC)},
		},
		{
			name: "respects name info in tag when tag has time format info",
			data: []string{"1988-11-08"},
			idx:  map[string]int{"date": 0},
			dst:  &nameAndFormatTaggedTimeThing{},
			want: &nameAndFormatTaggedTimeThing{time.Date(1988, time.November, 8, 0, 0, 0, 0, time.UTC)},
		},
		{
			name: "falls back to default assign function if assignFn returns ErrUseDefault",
			data: []string{"str", "1", "1.5"},
			idx:  nil,
			ass: map[reflect.Kind]AssignFn{ // use default for Int64 and Float64. String assignFn returns ErrUseDefault
				reflect.Int64:   defaultAssigners[reflect.Int64],
				reflect.Float64: defaultAssigners[reflect.Float64],
				reflect.String:  func(s string, v reflect.Value, t reflect.StructTag) error { return ErrUseDefault },
			},
			dst:  &thing{},
			want: &thing{"str", 1, 1.5},
		},
	}
	for _, test := range tests {
		if test.ass == nil {
			test.ass = defaultAssigners
		}
		err := unmarshall(test.data, test.idx, test.ass, test.dst)
		if !reflect.DeepEqual(test.err, err) {
			t.Errorf("%s: Got error '%v', want '%v'", test.name, err, test.err)
		}
		if !reflect.DeepEqual(test.want, test.dst) {
			t.Errorf("%s: Dst unmarshalled to %v, want %v", test.name, test.dst, test.want)
		}
	}
}

func TestAssignStruct(t *testing.T) {
	type timeObj struct {
		Time time.Time
	}
	timeTests := []struct {
		s    string
		fmt  string
		want time.Time
		err  bool
	}{
		{"2014-01-02", "2006-01-02", time.Date(2014, time.January, 2, 0, 0, 0, 0, time.UTC), false},
		{"2014/02/03 15:32:13", "2006/01/02 15:04:05", time.Date(2014, time.February, 3, 15, 32, 13, 0, time.UTC), false},
		{"", "2006/01/02", time.Time{}, false},
		{"wasd", "2006/01/02", time.Time{}, true},
	}
	for _, test := range timeTests {
		obj := timeObj{}
		tag := `csv:",` + test.fmt + `"`
		err := assignStruct(test.s, reflect.ValueOf(&obj).Elem().Field(0), reflect.StructTag(tag))
		if test.err {
			_, wantErr := time.Parse(test.fmt, test.s)
			if err == nil || wantErr.Error() != err.Error() {
				t.Errorf("%s, %s returned wrong error. Got %v, want %v", test.s, test.fmt, err, wantErr)
			}
			if !obj.Time.Equal(time.Time{}) {
				t.Errorf("assignStruct should return zero date on error. Got %v", obj.Time)
			}
		} else {
			if err != nil {
				t.Errorf("%s, %s returned error %v", test.s, test.fmt, err)
			} else {
				if !obj.Time.Equal(test.want) {
					t.Errorf("%s, %s sat time to wrong value. Got %v, want %v", test.s, test.fmt, test.want, obj.Time)
				}
			}
		}
	}
}
