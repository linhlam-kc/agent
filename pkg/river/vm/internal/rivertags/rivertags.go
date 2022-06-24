package rivertags

import (
	"fmt"
	"reflect"
	"strings"
)

type tagFlags uint

const (
	flagAttr  tagFlags = 1 << iota // Decode as block attribute
	flagBlock                      // Decode as block subblock
	flagKey                        // Decode as object field

	flagOptional // Optional field
	flagLabel    // Decode as block label
)

// Field is an tagged field within a struct.
type Field struct {
	Name  string // Name specified within river tag.
	Index int    // Field index in struct.

	flags tagFlags // Flags assigned to tag
}

// IsIgnored returns true if the tagged field is meant to be ignored.
func (f Field) IsIgnored() bool { return f.Name == "-" }

// IsAttr returns true if the tagged field is meant for decoding block
// attributes.
func (f Field) IsAttr() bool { return (f.flags & flagAttr) != 0 }

// IsBlock returns true if the tagged field is meant for decoding children
// blocks.
func (f Field) IsBlock() bool { return (f.flags & flagBlock) != 0 }

// IsKey returns true if the tagged field is meant for decoding object keys.
func (f Field) IsKey() bool { return (f.flags & flagKey) != 0 }

// IsOptional returns true if the tagged field is optional during decoding.
func (f Field) IsOptional() bool { return (f.flags & flagOptional) != 0 }

// IsLabel returns true if the tagged field is meant for decoding block labels.
func (f Field) IsLabel() bool { return (f.flags & flagLabel) != 0 }

// Fields is the list of tagged fields within a struct.
type Fields []Field

// BlockKind returns true if tagged fields are used for blocks.
func (ff Fields) BlockKind() bool {
	for _, tf := range ff {
		if tf.Name == "-" {
			continue
		}
		if tf.IsAttr() || tf.IsBlock() || tf.IsLabel() {
			return true
		}
	}
	return false
}

// ObjectKind returns true if tagged fields are used for objects.
func (ff Fields) ObjectKind() bool {
	for _, tf := range ff {
		if tf.Name == "-" {
			continue
		}
		if tf.IsKey() {
			return true
		}
	}
	return false
}

func (ff Fields) Get(name string) (f Field, ok bool) {
	for _, tf := range ff {
		if tf.Name == name {
			return tf, true
		}
	}
	return
}

// Get returns the list of tagged fields for some struct type. Panics if ty is
// not for a struct.
func Get(ty reflect.Type) Fields {
	if ty.Kind() != reflect.Struct {
		panic(fmt.Sprintf("rivertags: Get requires struct kind, got %s", ty.Kind()))
	}

	var (
		usedNames      = make(map[string]int)
		usedLabelField = -1
	)

	var res Fields

	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)

		tag, exists := field.Tag.Lookup("river")
		if !exists {
			continue
		}

		if tag == "-" {
			res = append(res, Field{
				Name:  "-",
				Index: i,
			})
			continue
		}

		frags := strings.SplitN(tag, ",", 2)
		if len(frags) == 0 {
			panic(fmt.Sprintf("river: unsupported empty tag in %s.%s", ty.String(), field.Name))
		}

		tf := Field{
			Name:  frags[0],
			Index: i,
		}

		if first, used := usedNames[tf.Name]; used && tf.Name != "" {
			panic(fmt.Sprintf("river: %s already used by %s.%s", tf.Name, ty.String(), ty.Field(first).Name))
		}
		usedNames[tf.Name] = i

		if len(frags) == 2 {
			switch frags[1] {
			case "attr":
				tf.flags = flagAttr
			case "attr,optional":
				tf.flags = flagAttr | flagOptional
			case "block":
				tf.flags = flagBlock
			case "block,optional":
				tf.flags = flagBlock | flagOptional
			case "key":
				tf.flags = flagKey
			case "key,optional":
				tf.flags = flagKey | flagOptional
			case "label":
				tf.flags = flagLabel
			default:
				panic(fmt.Sprintf("river: unsupported river tag format %q on %s.%s", tag, ty.String(), field.Name))
			}
		}

		if tf.IsLabel() {
			if usedLabelField >= 0 {
				panic(fmt.Sprintf("river: label field already used by %s.%s", ty.String(), ty.Field(usedLabelField).Name))
			}
			usedLabelField = i
		}

		if tf.Name == "" && (tf.IsBlock() || tf.IsAttr() || tf.IsKey()) {
			panic(fmt.Sprintf("river: non-empty field name required in %s.%s", ty.String(), ty.Field(i).Name))
		}

		res = append(res, tf)
	}

	if res.BlockKind() && res.ObjectKind() {
		panic(fmt.Sprintf("river: struct %s has tags for both objects and blocks at the same time", ty.String()))
	}

	return res
}
