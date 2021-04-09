package cmdutil

import (
	"fmt"
	"go/ast"
	"os"
	"reflect"
	"strings"

	"github.com/octohelm/kube-agent/pkg/reflectutil"
	"github.com/spf13/pflag"
)

func MustAddFlags(flagSet *pflag.FlagSet, v interface{}, envVarPrefix string) {
	if err := AddFlags(flagSet, v, envVarPrefix); err != nil {
		panic(err)
	}
}

func AddFlags(flags *pflag.FlagSet, v interface{}, envVarPrefix string) error {
	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(v)
	}

	kind := rv.Kind()
	if kind != reflect.Ptr {
		return fmt.Errorf("non-ptr reflectValue %v is not support", v)
	}

	rv = rv.Elem()

	if rv.Kind() != reflect.Struct {
		return nil
	}

	structTpe := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := structTpe.Field(i)

		flag := field.Name

		if !ast.IsExported(flag) {
			continue
		}

		if tag, ok := field.Tag.Lookup("flag"); ok {
			tv := reflectutil.NewTagValue(tag)
			if tv.IsIgnore() {
				continue
			}
			if na, ok := tv.Name(); ok {
				flag = na
			}

			v := reflectValue(rv.Field(i))

			desc := field.Tag.Get("desc")

			if defaultValue, ok := field.Tag.Lookup("default"); ok {
				if err := reflectutil.UnmarshalText(reflect.Value(v), []byte(defaultValue)); err != nil {
					panic(fmt.Errorf("invalid default value %s", defaultValue))
				}
			}

			_, ok := tv.LookupFlag("env")
			if ok {
				envVarKey := strings.ToUpper(envVarPrefix + "_" + strings.Replace(flag, "-", "_", -1))

				if envVarValue, ok := os.LookupEnv(envVarKey); ok {
					if err := reflectutil.UnmarshalText(reflect.Value(v), []byte(envVarValue)); err != nil {
						panic(fmt.Errorf("invalid value %s", envVarValue))
					}
				}

				d := fmt.Sprintf("${%s}", envVarKey)

				if desc != "" {
					desc = desc + " " + d
				} else {
					desc = d
				}

			}

			flags.VarP(v, flag, "", desc)
		}
	}

	return nil
}

type reflectValue reflect.Value

func (v reflectValue) Set(s string) error {
	return reflectutil.UnmarshalText(reflect.Value(v), []byte(s))
}

func (v reflectValue) String() string {
	return reflect.Value(v).String()
}

func (v reflectValue) Type() string {
	return reflect.Value(v).Type().String()
}
