package envstruct

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Envstruct struct {
	// Prefix is optional and if set, is used as the prefix to any environment
	// variable fetching. For example, if we are fetching env `FIELD1` and we
	// have prefix set to `BAR`, then `BAR_FIELD1` will be used to fetch the
	// environment variable.
	Prefix string

	// TagName is used for fetching the tag value from the field.
	TagName string

	// Parser includes the custom unmarshaler that will be used to unmarshal the
	// values into the fields. The only thing that envstruct does itself is unwrap
	// slices and maps but the underlying values within those types are parsed by
	// the unmarshaler.
	Parser Parser
}

func New(prefix, tagName string, parser Parser) Envstruct {
	return Envstruct{
		Prefix:  prefix,
		TagName: tagName,

		Parser: parser,
	}
}

// FetchEnv will fetch environment variables and appropriately set them into
// the struct given. The details on how the environemnt variables will be
// fetched is dictated by field tags. Nested tags are supported. It will
// overwrite the struct with any env values set.
func (e Envstruct) FetchEnv(object interface{}) error {
	// Check if the object is a struct
	if reflect.TypeOf(object).Elem().Kind() != reflect.Struct {
		return errors.New("failed to parse env into object, needs to be type struct")
	}

	// Uppercase the prefix value
	envPrefix := strings.ToUpper(e.Prefix)

	// Loop through each field within the struct
	v := reflect.ValueOf(object).Elem()
	for i := 0; i < v.NumField(); i++ {
		// Start building up the string that will be used to fetch the env. It
		// starts with the prefix (if set) and can contain any nested struct tag
		// values and field tag values.
		var envNameBuilder []string
		if e.Prefix != "" {
			envNameBuilder = []string{envPrefix}
		}

		// Extract the tag from the field value and use it to fetch the env into
		// the struct
		err := e.extractTag(envNameBuilder, v.Type().Field(i), v.Field(i))
		if err != nil {
			return err
		}
	}

	return nil
}

func (e Envstruct) extractTag(envNameBuilder []string, fieldDescription reflect.StructField, fieldValue reflect.Value) error {
	// Fetch the tag value from the struct and append it to the string that will
	// be used to fetch the env value
	tagValue := fieldDescription.Tag.Get(e.TagName)
	if tagValue != "" {
		envNameBuilder = append(envNameBuilder, strings.ToUpper(tagValue))
	}

	// If the field is a struct then loop through each field and recurse
	if fieldDescription.Type.Kind() == reflect.Struct {
		for i := 0; i < fieldValue.NumField(); i++ {
			err := e.extractTag(envNameBuilder, fieldValue.Type().Field(i), fieldValue.Field(i))
			if err != nil {
				return err
			}
		}
	} else {
		// If the field is not a struct, fetch the environment variable value using
		// the built up string
		value := os.Getenv(strings.Join(envNameBuilder, "_"))

		// Parse the fetched env value and set it on the field
		err := e.Parser.ParseInto(fieldValue.Addr().Interface(), value)
		if err != nil {
			return err
		}
	}

	return nil
}

type Parser struct {
	// Delimiter is used as the separater for multiple values within a struct or
	// map. It is defaulted to a comma ",". It is used so that in the environment
	// variable, there can exist slices such as "PREFIX_FIELD=foo,bar".
	Delimiter string

	Unmarshaler UnmarshalFunc
}

type UnmarshalFunc func([]byte, interface{}) error

// ParseInto will parse the value given into the fieldValue. If the value is a
// slice or a map, it will manually separate each item within the array of
// items and pass them to the unmarshaler. If not, the value will be directly
// passed to the unmarshaller.
//
// IMPORTANT: It currently DOES NOT SUPPORT NESTED SLICES OR MAPS. For ex,
// "[][]string" will not be parsed correctly.
func (p Parser) ParseInto(fieldValue interface{}, value string) error {
	if p.Unmarshaler == nil {
		return errors.New("no unmarshaler set for parser")
	}

	// Default delimiter is comma
	delimiter := ","
	if p.Delimiter != "" {
		delimiter = p.Delimiter
	}

	fieldType := reflect.TypeOf(fieldValue).Elem()

	// Two special types of fields that we have to manually parse is a slice and
	// a map. XXX: Will we ever need to parse nested slices/maps?
	switch fieldType.Kind() {
	case reflect.Slice:
		// Split the field value into separate elements in a slice
		envSlice := strings.Split(fmt.Sprintf("%v", value), delimiter)

		// Make an empty slice that is the same type as the field in the struct
		unmarshalledSlice := reflect.MakeSlice(fieldType, 0, 0)

		// Loop through each element within the split string
		for _, s := range envSlice {
			// Create a variable that is the same type of the individual slice
			// elements
			elem := reflect.New(fieldType.Elem())

			// Unmarshal the env into the interface of the element
			err := p.Unmarshaler([]byte(strings.TrimSpace(s)), elem.Interface())
			if err != nil {
				return err
			}

			// Append each unmarshalled value into the unmarshalled slice. When
			// appending the element, we want to append the value of the element
			// rather than a pointer type, which is why we use Elem() to dereference
			// it.
			unmarshalledSlice = reflect.Append(unmarshalledSlice, elem.Elem())
		}

		// Set the unmarshalled slice onto the slice struct field
		reflect.ValueOf(fieldValue).Elem().Set(unmarshalledSlice)

	case reflect.Map:
		// Split the field value into separate key,value pairs in a map
		envMap := strings.Split(fmt.Sprintf("%v", value), delimiter)

		// Make an empty map that is the same type as the field in the struct
		unmarshalledMap := reflect.MakeMap(fieldType)
		for _, envPair := range envMap {
			// Split the map into the key and value
			keyVal := strings.Split(fmt.Sprintf("%v", envPair), ":")
			if len(keyVal) > 2 {
				return errors.New(fmt.Sprintf("failed to parse map value %v", envPair))
			}

			// Create a variable that is the same type of the key type
			key := reflect.New(fieldType.Key())

			// Unmarshal the env into the key variable
			err := p.Unmarshaler([]byte(strings.TrimSpace(keyVal[0])), key.Interface())
			if err != nil {
				return err
			}

			// Create a variable that is the same type of the value type
			value := reflect.New(fieldType.Elem())

			// Unmarshal the env into the value variable
			err = p.Unmarshaler([]byte(strings.TrimSpace(keyVal[1])), value.Interface())
			if err != nil {
				return err
			}

			// Set the key and value on the unmarshalled map. When setting the key
			// value pairs, we want to set the value of the pair rather than a
			// pointer type, which is why we use Elem() to dereference it.
			unmarshalledMap.SetMapIndex(key.Elem(), value.Elem())
		}

		// Set the unmarshalled map onto the map struct field
		reflect.ValueOf(fieldValue).Elem().Set(unmarshalledMap)
	default:
		err := p.Unmarshaler([]byte(value), fieldValue)
		if err != nil {
			return err
		}
	}

	return nil
}