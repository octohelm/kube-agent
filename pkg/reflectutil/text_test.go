package reflectutil

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-courier/ptr"
	. "github.com/onsi/gomega"
)

type Duration time.Duration

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

func (d *Duration) UnmarshalText(data []byte) error {
	dur, err := time.ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

type NamedString string

func TestMarshalTextAndUnmarshalText(t *testing.T) {
	v := struct {
		NamedString NamedString
		Duration    Duration
		PtrDuration *Duration
		String      string
		PtrString   *string
		Int         int
		PtrInt      *int
		Uint        uint
		PtrUint     *uint
		Float       float32
		PtrFloat    *float32
		Bool        bool
		PtrBool     *bool
	}{}

	rv := reflect.ValueOf(&v).Elem()

	d := Duration(2 * time.Second)

	cases := []struct {
		v      interface{}
		text   string
		expect interface{}
	}{
		{
			rv.FieldByName("NamedString"),
			"string",
			NamedString("string"),
		},
		{
			rv.FieldByName("PtrString"),
			"string",
			ptr.String("string"),
		},
		{
			&v.String,
			"ptr",
			ptr.String("ptr"),
		},
		{
			rv.FieldByName("Duration"),
			"2s",
			Duration(2 * time.Second),
		},
		{
			rv.FieldByName("PtrDuration"),
			"2s",
			&d,
		},
		{
			rv.FieldByName("PtrString"),
			"string",
			ptr.String("string"),
		},
		{
			rv.FieldByName("Int"),
			"1",
			1,
		},
		{
			rv.FieldByName("PtrInt"),
			"1",
			ptr.Int(1),
		},
		{
			rv.FieldByName("Uint"),
			"1",
			uint(1),
		},
		{
			rv.FieldByName("PtrUint"),
			"1",
			ptr.Uint(1),
		},
		{
			rv.FieldByName("Float"),
			"1",
			float32(1),
		},
		{
			rv.FieldByName("PtrFloat"),
			"1",
			ptr.Float32(1),
		},
		{
			rv.FieldByName("Bool"),
			"true",
			true,
		},
		{
			rv.FieldByName("PtrBool"),
			"true",
			ptr.Bool(true),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d UnmarshalText %v", i, c.v), func(t *testing.T) {
			err := UnmarshalText(c.v, []byte(c.text))

			NewWithT(t).Expect(err).To(BeNil())

			if rv, ok := c.v.(reflect.Value); ok {
				NewWithT(t).Expect(c.expect).To(Equal(rv.Interface()))
			} else {
				NewWithT(t).Expect(c.expect).To(Equal(c.v))
			}
		})
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d MarshalText by %v", i, c.text), func(t *testing.T) {
			text, err := MarshalText(c.v)
			NewWithT(t).Expect(err).To(BeNil())
			NewWithT(t).Expect(c.text).To(Equal(string(text)))
		})
	}

	v2 := struct {
		PtrString *string
		Slice     []string
	}{}

	rv2 := reflect.ValueOf(v2)

	{
		_, err := MarshalText(rv2.FieldByName("Slice"))
		NewWithT(t).Expect(err).NotTo(BeNil())
	}

	{
		_, err := MarshalText(rv2.FieldByName("PtrString"))
		NewWithT(t).Expect(err).To(BeNil())
	}
}
