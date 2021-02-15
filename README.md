# envstruct

Library to parse env into a struct

## What is envstruct

It supports all basic parsing of environment variables into the struct object
given. An example of it's usage is that you would pass in the following struct:

```go
type MyStruct struct {
  Field string `tag:"field"`
}
```

`envstruct` will then fetch the environment variable `FIELD`, parse it and set it
back into your struct. So if you had `FIELD=foo` set as an envionment variable,
the result will be:

```go
MyStruct{
  Field: "foo"
}
```

I started this project in order to satisfy my use case of being able to use a
library that will parse environment variables into a struct using tag values,
including the tag values of nested structs. There are a lot of libraries out
there that can parse environment variables into structs, but the environment
variable that it fetches was built up either using the field names or field tag
values that don't include tag values on nested structs.

For example, it supports the following struct:

```go
type MyStruct struct {
  Nested struct {
    Field string `tag:"field"`
  } `tag:"nested"`
}
```

Which the `Field` value will be fetched using `NESTED_FIELD` as the string to
fetch the environment variable.

## How to use it

You can use `envstruct` by first configuring a few settings.

| Settings      | Desciptions           
| ------------- |-------------
| Prefix        | Optional and if set, is used as the prefix to any environment variable fetching. For example, if we are fetching env string `FIELD1` and we have prefix set to `BAR`, then `BAR_FIELD1` will be used to fetch the environment variable.
| TagName       | Used for fetching the tag value from the field. A built up string using this tag value will be used to fetch the environment variable. Can be placed on a struct or field.
| Delimiter     | Used as the separater for multiple values within a struct or map. It is defaulted to a comma `,`. It is used so that in the environment variable, there can exist slices such as `PREFIX_FIELD=foo,bar`.
| Unmarshaler   | Used to unmarshal the string into the field types. For example, you can pass in a `yaml` or `json` unmarshaler.
| OverrideName  | Optional and if set, is used to fetch the tag value from the field that will be used to fetch the environment variable. It is used to override the string built using the `TagName`. The tag value from `OverrideName` will be used directly and will not be modified with upper casing, prefixing or attaching nested struct tag values.

Then you call `FetchEnv` off of `envstruct`.

```go
env := envstruct.New("prefix", "tag", envstruct.Parser{Delimiter: ",", Unmarshaler: yaml.Unmarshal})

type Example struct {
  Field string `tag:"field"`
}

mystruct := &Example{}

err := env.FetchEnv(mystruct)
if err != nil {
  return nil
}
```

From the example above, if the environment variable `PREFIX_FIELD=foo` was set
while it was running then `mystruct.Field` will be populated with the string
`foo`.

### How are environment variables parsed

The types of variables that it parses depends on what kind of `Unmarshaler` is
passed to the `envstruct`. The only special cased types are **slices** and
**maps**, which are parsed by `envstruct` and then each item is passed to the
`Unmarshaler`. Multiple items within one environment variables are separated by
the `Delimiter` that is set on the `envstruct`.

An example of a slice would be:

```
PREFIX_SLICE=foo,bar
```

An example of a map would be:

```
PREFIX_MAP=foo:foo1,bar:bar1
```

Each part of the string that is used to build up the environment variable is
uppercased and appended to each other using an underscore `_`.

### How is the struct parsed

The exact string that is used to fetch the environment variable is built up
using the tags within the struct fields. The `TagName` will be used to fetch
the tag value from the struct fields for building up the environment variable
string.

A simple example is a basic struct with a field:

```go
type MyStruct struct {
  FieldName string `tag:"field"`
}
```

`FIELD` will be the string that is used to fetch the value of the environment
variable for `FieldName`.

If the field does not have a tag that matches the `TagName`, it will not be
fetched from an environment variable. `envstruct` will only fetch fields that
has a tag that matches the `TagName`.

Fields within nested structs are supported, and the environment variable string
is built up with the tag values from each nested struct that has a tag matching
the `TagName`.

For example,

```go
type MyStruct struct {
  Foo struct {
    Bar struct {
      FieldName string `tag:"field"`
    }
  } `tag:"foo"`
}
```

The example above would result in the string `FOO_BAR` to be used to fetch the
environment variable for `MyStruct.Foo.Bar.FieldName`. If the `Prefix` was set
to `PREFIX`, then `PREFIX_FOO_BAR` will be used to fetch the environment
variable for `MyStruct.Foo.Bar.FieldName`.

## Overriding the tag

The string that is built up using the `TagName` which is used to fetch the
environment variable can be overriden using the `OverrideName` field. If you
configure the `OverrideName` field, any field that contains a tag that matches
the value set in `OverrideName` will use the value of that tag to fetch its
environment variable.

The value of the tag matching the `OverrideName` will be used directly, that
means it will not be uppercased, appended with nested struct tags or prefixed
like how the regular built up string using the `TagName` is.

For example, in the example below we have set the `OverrideName` to be value
`override`.

```go
type MyStruct struct {
  Foo struct {
    Bar struct {
      FieldName string `tag:"field" override:"override_field"`
    }
  } `tag:"foo"`
}
```

Since `FieldName` has a tag `override`, the value of the `override` value will
be used to fetch the environment variable. In the example, `override_field`
will be used to fetch the environment variable, rather than `FOO_BAR` which
would have been used if the field did not contain the `override` tag.

A field can have multiple override tag values that are comma separated. For
example, 

```go
type MyStruct struct {
  FieldName string `tag:"field" override:"O_FIELD1,O_FIELD2"`
}
```

`O_FIELD1` and `O_FIELD2` will be used to try and fetch the environment
variable for `FieldName`. It is ordered in terms of precedence from left to
right, so if a value is fetched from `O_FIELD1` then we will use that value and
not try to fetch using `O_FIELD2`.
