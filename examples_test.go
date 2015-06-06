package csvdecoder

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// The example shows how to decode to a struct using the struct fields' position within the struct.
// The Name and Age fields are at index 0 and 1, respectively, within the Student struct, matching
// the position of the data in the input string.
func ExampleDecoder_Decode_simple() {
	type Student struct {
		Name string
		Age  int
	}

	const input = "John,21\nSusan,24"
	r := csv.NewReader(strings.NewReader(input))
	decoder := NewDecoder(r)
	for {
		var student Student
		if err := decoder.Decode(&student); err == io.EOF {
			break
		} else if err != nil {
			fmt.Errorf("%v", err)
			return
		}
		fmt.Printf("Name: %s, Age: %d\n", student.Name, student.Age)
	}
	// Output: Name: John, Age: 21
	// Name: Susan, Age: 24
}

// The example shows how to read a header before parsing the input. The values in the header
// corresponds with the field names of the struct.
func ExampleDecoder_Decode_fieldNames() {
	type Student struct {
		Name string
		Age  int
	}
	const input = "Age,Name\n21,John\n24,Susan"
	r := csv.NewReader(strings.NewReader(input))
	decoder := NewDecoder(r)

	// Read the first line as a header
	if err := decoder.ReadHeader(); err != nil {
		fmt.Errorf("%v", err)
		return
	}
	for {
		var student Student
		if err := decoder.Decode(&student); err == io.EOF {
			break
		} else if err != nil {
			fmt.Errorf("%v", err)
			return
		}
		fmt.Printf("Name: %s, Age: %d\n", student.Name, student.Age)
	}
	// Output: Name: John, Age: 21
	// Name: Susan, Age: 24
}

// The example shows how to explicitly specify the indexes of each field before parsing
// the input
func ExampleDecoder_Decode_indexes() {
	type Student struct {
		Name string
		Age  int
	}
	const input = "21,John\n24,Susan"
	r := csv.NewReader(strings.NewReader(input))
	decoder := NewDecoder(r)

	// Set the indexes
	decoder.Indexes = map[string]int{
		"Age":  0,
		"Name": 1,
	}
	for {
		var student Student
		if err := decoder.Decode(&student); err == io.EOF {
			break
		} else if err != nil {
			fmt.Errorf("%v", err)
			return
		}
		fmt.Printf("Name: %s, Age: %d\n", student.Name, student.Age)
	}
	// Output: Name: John, Age: 21
	// Name: Susan, Age: 24
}

// The example shows how to use attributes to specify the field name values to look
// for when reading the header.
func ExampleDecoder_Decode_attributes() {
	type Student struct {
		Name string `csv:"name"`
		Age  int    `csv:"age"`
	}
	const input = "name,age\nJohn,21\nSusan,24"
	r := csv.NewReader(strings.NewReader(input))
	decoder := NewDecoder(r)

	// Read the first line as a header, using the attributes
	if err := decoder.ReadHeader(); err != nil {
		fmt.Errorf("%v", err)
		return
	}
	for {
		var student Student
		if err := decoder.Decode(&student); err == io.EOF {
			break
		} else if err != nil {
			fmt.Errorf("%v", err)
			return
		}
		fmt.Printf("Name: %s, Age: %d\n", student.Name, student.Age)
	}
	// Output: Name: John, Age: 21
	// Name: Susan, Age: 24
}

// The example shows how to use attributes to specify the time format to parse datetime values
func ExampleDecoder_Decode_time() {
	type Student struct {
		Name     string
		Birthday time.Time `csv:",2006-01-02"`
	}
	const input = "John,1994-05-14\nSusan,1991-12-03"
	r := csv.NewReader(strings.NewReader(input))
	decoder := NewDecoder(r)

	for {
		var student Student
		if err := decoder.Decode(&student); err == io.EOF {
			break
		} else if err != nil {
			fmt.Errorf("%v", err)
			return
		}
		fmt.Printf("Name: %s, Birthday: %s\n", student.Name, student.Birthday.Format("Jan 2, 2006"))
	}
	// Output: Name: John, Birthday: May 14, 1994
	// Name: Susan, Birthday: Dec 3, 1991
}
