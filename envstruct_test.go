package envstruct_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/clarafu/envstruct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

func TestEnvstruct(t *testing.T) {
	suite.Run(t, &EnvstructSuite{
		Assertions: require.New(t),
	})
}

type EnvstructSuite struct {
	suite.Suite
	*require.Assertions
}

type EnvstructTest struct {
	It string

	Prefix        string
	TagName       string
	OverrideName  string
	IgnoreTagName string
	Delimiter     string
	StripValue    bool

	EnvValues map[string]interface{}

	TestStruct   interface{}
	ResultStruct interface{}
}

func createString(x string) *string {
	return &x
}

func (s *EnvstructSuite) TestEnvstruct() {
	for _, t := range []EnvstructTest{
		{
			It: "parses env into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "value",
				"PREFIX_FIELD3": 3,
			},

			TestStruct: &struct {
				Field1 string `tag:"field1"`
				Field2 string
				Field3 int `tag:"field3"`
			}{},

			ResultStruct: &struct {
				Field1 string `tag:"field1"`
				Field2 string
				Field3 int `tag:"field3"`
			}{
				Field1: "value",
				Field3: 3,
			},
		},
		{
			It: "continues to fetch fields even if env is not found",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD2": "value",
			},

			TestStruct: &struct {
				Field1 string `tag:"field1"`
				Field2 string `tag:"field2"`
			}{},

			ResultStruct: &struct {
				Field1 string `tag:"field1"`
				Field2 string `tag:"field2"`
			}{
				Field2: "value",
			},
		},
		{
			It: "parses uncommon types into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "true",
				"PREFIX_FIELD2": "1.23",
				"PREFIX_FIELD3": "1234",
				"PREFIX_FIELD4": "1h",
				"PREFIX_FIELD5": "hi",
			},

			TestStruct: &struct {
				Field1 bool          `tag:"field1"`
				Field2 float64       `tag:"field2"`
				Field3 uint16        `tag:"field3"`
				Field4 time.Duration `tag:"field4"`
				Field5 *string       `tag:"field5"`
			}{},

			ResultStruct: &struct {
				Field1 bool          `tag:"field1"`
				Field2 float64       `tag:"field2"`
				Field3 uint16        `tag:"field3"`
				Field4 time.Duration `tag:"field4"`
				Field5 *string       `tag:"field5"`
			}{
				Field1: true,
				Field2: 1.23,
				Field3: uint16(1234),
				Field4: 1 * time.Hour,
				Field5: createString("hi"),
			},
		},
		{
			It: "parses slices into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "value,value2",
				"PREFIX_FIELD2": "1,2",
			},

			TestStruct: &struct {
				Field1 []string `tag:"field1"`
				Field2 []int    `tag:"field2"`
			}{},

			ResultStruct: &struct {
				Field1 []string `tag:"field1"`
				Field2 []int    `tag:"field2"`
			}{
				Field1: []string{"value", "value2"},
				Field2: []int{1, 2},
			},
		},
		{
			It: "parses slices into struct, removing surrounding spaces but persisting spaces within value",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": " value , value 2 ",
			},

			TestStruct: &struct {
				Field1 []string `tag:"field1"`
			}{},

			ResultStruct: &struct {
				Field1 []string `tag:"field1"`
			}{
				Field1: []string{"value", "value 2"},
			},
		},
		{
			It: "parses slices into struct with custom delimter",

			Prefix:    "prefix",
			TagName:   "tag",
			Delimiter: ":",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "1:2",
			},

			TestStruct: &struct {
				Field1 []int `tag:"field1"`
			}{},

			ResultStruct: &struct {
				Field1 []int `tag:"field1"`
			}{
				Field1: []int{1, 2},
			},
		},
		{
			It: "parses types into struct using the designated field type rather than the type of the value itself",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "1,2",
			},

			TestStruct: &struct {
				Field1 []int64 `tag:"field1"`
			}{},

			ResultStruct: &struct {
				Field1 []int64 `tag:"field1"`
			}{
				Field1: []int64{1, 2},
			},
		},
		{
			It: "parses maps into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "key:value,key2:value2",
				"PREFIX_FIELD2": "1:2",
			},

			TestStruct: &struct {
				Field1 map[string]string `tag:"field1"`
				Field2 map[int]int       `tag:"field2"`
			}{},

			ResultStruct: &struct {
				Field1 map[string]string `tag:"field1"`
				Field2 map[int]int       `tag:"field2"`
			}{
				Field1: map[string]string{"key": "value", "key2": "value2"},
				Field2: map[int]int{1: 2},
			},
		},
		{
			It: "parses maps into struct, removing surrounding spaces but persisting spaces within value",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": " key : value ,key 2: value 2",
			},

			TestStruct: &struct {
				Field1 map[string]string `tag:"field1"`
			}{},

			ResultStruct: &struct {
				Field1 map[string]string `tag:"field1"`
			}{
				Field1: map[string]string{"key": "value", "key 2": "value 2"},
			},
		},
		{
			It: "parses nested env without tag name into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "value",
				"PREFIX_FIELD2": "nestedvalue",
			},

			TestStruct: &struct {
				Field1      string `tag:"field1"`
				NestedField struct {
					Field2 string `tag:"field2"`
				}
			}{},

			ResultStruct: &struct {
				Field1      string `tag:"field1"`
				NestedField struct {
					Field2 string `tag:"field2"`
				}
			}{
				Field1: "value",
				NestedField: struct {
					Field2 string `tag:"field2"`
				}{
					Field2: "nestedvalue",
				},
			},
		},
		{
			It: "parses nested env with tag name into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1":        "value",
				"PREFIX_NESTED_FIELD2": "nestedvalue",
			},

			TestStruct: &struct {
				Field1      string `tag:"field1"`
				NestedField struct {
					Field2 string `tag:"field2"`
				} `tag:"nested"`
			}{},

			ResultStruct: &struct {
				Field1      string `tag:"field1"`
				NestedField struct {
					Field2 string `tag:"field2"`
				} `tag:"nested"`
			}{
				Field1: "value",
				NestedField: struct {
					Field2 string `tag:"field2"`
				}{
					Field2: "nestedvalue",
				},
			},
		},
		{
			It: "parses multi nested env with tag name into struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_NESTED_NESTED2_FIELD": "nestedvalue",
			},

			TestStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2"`
				} `tag:"nested"`
			}{},

			ResultStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2"`
				} `tag:"nested"`
			}{
				NestedField: struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2"`
				}{
					NestedField2: struct {
						Field2 string `tag:"field"`
					}{
						Field2: "nestedvalue",
					},
				},
			},
		},
		{
			It: "only uses tagged struct names in multi nested struct",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_NESTED2_FIELD": "nestedvalue",
			},

			TestStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2"`
				}
			}{},

			ResultStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2"`
				}
			}{
				NestedField: struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2"`
				}{
					NestedField2: struct {
						Field2 string `tag:"field"`
					}{
						Field2: "nestedvalue",
					},
				},
			},
		},
		{
			It: "uses the override tag name with no prefix or nesting to fetch the env",

			Prefix:       "prefix",
			TagName:      "tag",
			OverrideName: "override",

			EnvValues: map[string]interface{}{
				"PREFIX_NESTED_NESTED2_FIELD": "nestedvalue",
				"override_field":              "overridevalue",
			},

			TestStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field" override:"override_field"`
					} `tag:"nested2"`
				} `tag:"nested"`
			}{},

			ResultStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field" override:"override_field"`
					} `tag:"nested2"`
				} `tag:"nested"`
			}{
				NestedField: struct {
					NestedField2 struct {
						Field2 string `tag:"field" override:"override_field"`
					} `tag:"nested2"`
				}{
					NestedField2: struct {
						Field2 string `tag:"field" override:"override_field"`
					}{
						Field2: "overridevalue",
					},
				},
			},
		},
		{
			It: "tries to fetch env with multiple override tag names, first override tag takes precendence",

			Prefix:       "prefix",
			TagName:      "tag",
			OverrideName: "override",

			EnvValues: map[string]interface{}{
				"OVERRIDE_FIELD1_CAPS": "field1value",
				"override_field2":      "first",
				"OVERRIDE_FIELD2_CAPS": "second",
			},

			TestStruct: &struct {
				Field1 string `tag:"field1" override:"override_field1,OVERRIDE_FIELD1_CAPS"`
				Field2 string `tag:"field2" override:"override_field2,OVERRIDE_FIELD2_CAPS"`
			}{},

			ResultStruct: &struct {
				Field1 string `tag:"field1" override:"override_field1,OVERRIDE_FIELD1_CAPS"`
				Field2 string `tag:"field2" override:"override_field2,OVERRIDE_FIELD2_CAPS"`
			}{
				Field1: "field1value",
				Field2: "first",
			},
		},
		{
			It: "ignores tags that are labelled with ignore tag name",

			Prefix:        "prefix",
			TagName:       "tag",
			IgnoreTagName: "ignore",

			EnvValues: map[string]interface{}{
				"PREFIX_NESTED_FIELD": "ignoretrue",
			},

			TestStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2" ignore:"true"`
				} `tag:"nested"`
			}{},

			ResultStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2" ignore:"true"`
				} `tag:"nested"`
			}{
				NestedField: struct {
					NestedField2 struct {
						Field2 string `tag:"field"`
					} `tag:"nested2" ignore:"true"`
				}{
					NestedField2: struct {
						Field2 string `tag:"field"`
					}{
						Field2: "ignoretrue",
					},
				},
			},
		},
		{
			It: "applies tags that are labelled with ignore false",

			Prefix:        "prefix",
			TagName:       "tag",
			IgnoreTagName: "ignore",

			EnvValues: map[string]interface{}{
				"PREFIX_NESTED_NESTED2": "ignorefalse",
			},

			TestStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field" ignore:"true"`
					} `tag:"nested2" ignore:"false"`
				} `tag:"nested"`
			}{},

			ResultStruct: &struct {
				NestedField struct {
					NestedField2 struct {
						Field2 string `tag:"field" ignore:"true"`
					} `tag:"nested2" ignore:"false"`
				} `tag:"nested"`
			}{
				NestedField: struct {
					NestedField2 struct {
						Field2 string `tag:"field" ignore:"true"`
					} `tag:"nested2" ignore:"false"`
				}{
					NestedField2: struct {
						Field2 string `tag:"field" ignore:"true"`
					}{
						Field2: "ignorefalse",
					},
				},
			},
		},
		{
			It: "strips value after comma if StripValue is set to true",

			Prefix:     "prefix",
			TagName:    "tag",
			StripValue: true,

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1": "value",
				"PREFIX_FIELD3": 3,
				"PREFIX_FIELD4": "nested_value",
			},

			TestStruct: &struct {
				Field1      string `tag:"field1,omitempty"`
				Field2      string
				Field3      int `tag:"field3"`
				NestedField struct {
					Field4 string `tag:"field4,omitempty"`
				} `tag:",inline"`
			}{},

			ResultStruct: &struct {
				Field1      string `tag:"field1,omitempty"`
				Field2      string
				Field3      int `tag:"field3"`
				NestedField struct {
					Field4 string `tag:"field4,omitempty"`
				} `tag:",inline"`
			}{
				Field1: "value",
				Field3: 3,
				NestedField: struct {
					Field4 string `tag:"field4,omitempty"`
				}{
					Field4: "nested_value",
				},
			},
		},
		{
			It: "does not strip value if StripValue is false",

			Prefix:  "prefix",
			TagName: "tag",

			EnvValues: map[string]interface{}{
				"PREFIX_FIELD1,OMITEMPTY": "value",
			},

			TestStruct: &struct {
				Field1 string `tag:"field1,omitempty"`
			}{},

			ResultStruct: &struct {
				Field1 string `tag:"field1,omitempty"`
			}{
				Field1: "value",
			},
		},
	} {
		s.Run(t.It, func() {
			env := envstruct.Envstruct{
				Prefix:        t.Prefix,
				TagName:       t.TagName,
				OverrideName:  t.OverrideName,
				IgnoreTagName: t.IgnoreTagName,
				StripValue:    t.StripValue,

				Parser: envstruct.Parser{Delimiter: t.Delimiter, Unmarshaler: yaml.Unmarshal},
			}

			for name, value := range t.EnvValues {
				os.Setenv(name, fmt.Sprintf("%v", value))
			}

			err := env.FetchEnv(t.TestStruct)
			s.NoError(err)

			assert.Equal(s.T(), t.TestStruct, t.ResultStruct, "the struct should have correct env values populated")

			os.Clearenv()
		})
	}
}
