package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"
)

type stringListFlag []string

func (s *stringListFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type typedValueFlags struct {
	label        string
	valueText    *string
	stringValue  *string
	intValue     *string
	floatValue   *string
	boolValue    *string
	nodePath     *string
	vector2      *string
	vector3      *string
	color        *string
	resource     *string
	aabb         *string
	arrayVector2 *string
	arrayVector3 *string
	arrayString  *string
	arrayInt     *string
	arrayFloat   *string
	arrayBool    *string
}

func newTypedValueFlags(fs *flag.FlagSet, label string) *typedValueFlags {
	flags := &typedValueFlags{label: label}
	flags.valueText = fs.String("value", "", `typed JSON value, for example {"kind":"Vector2","value":[200,400]}`)
	flags.stringValue = fs.String("string", "", "string value shorthand")
	flags.intValue = fs.String("int", "", "integer value shorthand")
	flags.floatValue = fs.String("float", "", "float value shorthand")
	flags.boolValue = fs.String("bool", "", "boolean value shorthand")
	flags.nodePath = fs.String("node-path", "", "NodePath shorthand as /root/Node/Child")
	flags.vector2 = fs.String("vector2", "", "Vector2 shorthand as x,y")
	flags.vector3 = fs.String("vector3", "", "Vector3 shorthand as x,y,z")
	flags.color = fs.String("color", "", "Color shorthand as r,g,b[,a]")
	flags.resource = fs.String("resource", "", "Resource shorthand as res://path")
	flags.aabb = fs.String("aabb", "", "AABB shorthand as px,py,pz,sx,sy,sz (position x,y,z then size x,y,z)")
	flags.arrayVector2 = fs.String("array-vector2", "", "Array[Vector2] shorthand as x,y;x,y")
	flags.arrayVector3 = fs.String("array-vector3", "", "Array[Vector3] shorthand as x,y,z;x,y,z")
	flags.arrayString = fs.String("array-string", "", "Array[String] shorthand as a;b;c")
	flags.arrayInt = fs.String("array-int", "", "Array[int] shorthand as 1;2;3")
	flags.arrayFloat = fs.String("array-float", "", "Array[float] shorthand as 1.2;3.4")
	flags.arrayBool = fs.String("array-bool", "", "Array[bool] shorthand as true;false")
	return flags
}

func (f *typedValueFlags) Value() (any, error) {
	values := []struct {
		name  string
		value string
	}{
		{"value", *f.valueText},
		{"string", *f.stringValue},
		{"int", *f.intValue},
		{"float", *f.floatValue},
		{"bool", *f.boolValue},
		{"node-path", *f.nodePath},
		{"vector2", *f.vector2},
		{"vector3", *f.vector3},
		{"color", *f.color},
		{"resource", *f.resource},
		{"aabb", *f.aabb},
		{"array-vector2", *f.arrayVector2},
		{"array-vector3", *f.arrayVector3},
		{"array-string", *f.arrayString},
		{"array-int", *f.arrayInt},
		{"array-float", *f.arrayFloat},
		{"array-bool", *f.arrayBool},
	}
	count := 0
	for _, item := range values {
		if item.value != "" {
			count++
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("%s requires a value flag: --value, --string, --int, --float, --bool, --node-path, --vector2, --vector3, --color, --resource, --aabb, or --array-*", f.label)
	}
	if count > 1 {
		return nil, fmt.Errorf("%s requires exactly one value flag", f.label)
	}
	if *f.valueText != "" {
		var value any
		if err := json.Unmarshal([]byte(*f.valueText), &value); err != nil {
			return nil, fmt.Errorf("%s --value must be typed JSON: %w", f.label, err)
		}
		return value, nil
	}
	if *f.stringValue != "" {
		return map[string]any{"kind": "String", "value": *f.stringValue}, nil
	}
	if *f.intValue != "" {
		v, err := strconv.Atoi(*f.intValue)
		if err != nil {
			return nil, fmt.Errorf("%s --int must be an integer: %w", f.label, err)
		}
		return map[string]any{"kind": "int", "value": v}, nil
	}
	if *f.floatValue != "" {
		v, err := strconv.ParseFloat(*f.floatValue, 64)
		if err != nil {
			return nil, fmt.Errorf("%s --float must be a number: %w", f.label, err)
		}
		return map[string]any{"kind": "float", "value": v}, nil
	}
	if *f.boolValue != "" {
		v, err := strconv.ParseBool(*f.boolValue)
		if err != nil {
			return nil, fmt.Errorf("%s --bool must be true or false: %w", f.label, err)
		}
		return map[string]any{"kind": "bool", "value": v}, nil
	}
	if *f.nodePath != "" {
		return map[string]any{"kind": "NodePath", "value": *f.nodePath}, nil
	}
	if *f.vector2 != "" {
		values, err := parseFloatList(*f.vector2, 2, "vector2")
		if err != nil {
			return nil, fmt.Errorf("%s --vector2 %w", f.label, err)
		}
		return map[string]any{"kind": "Vector2", "value": values}, nil
	}
	if *f.vector3 != "" {
		values, err := parseFloatList(*f.vector3, 3, "vector3")
		if err != nil {
			return nil, fmt.Errorf("%s --vector3 %w", f.label, err)
		}
		return map[string]any{"kind": "Vector3", "value": values}, nil
	}
	if *f.color != "" {
		parts := strings.Split(*f.color, ",")
		if len(parts) != 3 && len(parts) != 4 {
			return nil, fmt.Errorf("%s --color must be r,g,b or r,g,b,a", f.label)
		}
		values, err := parseFloatList(*f.color, len(parts), "color")
		if err != nil {
			return nil, fmt.Errorf("%s --color %w", f.label, err)
		}
		return map[string]any{"kind": "Color", "value": values}, nil
	}
	if *f.aabb != "" {
		vals, err := parseFloatList(*f.aabb, 6, "aabb")
		if err != nil {
			return nil, fmt.Errorf("%s --aabb must be px,py,pz,sx,sy,sz: %w", f.label, err)
		}
		return map[string]any{
			"kind": "AABB",
			"value": map[string]any{
				"position": vals[:3],
				"size":     vals[3:],
			},
		}, nil
	}
	if *f.arrayVector2 != "" {
		values, err := parseVectorArray(*f.arrayVector2, 2, "array-vector2")
		if err != nil {
			return nil, fmt.Errorf("%s --array-vector2 %w", f.label, err)
		}
		return map[string]any{"kind": "Array[Vector2]", "value": values}, nil
	}
	if *f.arrayVector3 != "" {
		values, err := parseVectorArray(*f.arrayVector3, 3, "array-vector3")
		if err != nil {
			return nil, fmt.Errorf("%s --array-vector3 %w", f.label, err)
		}
		return map[string]any{"kind": "Array[Vector3]", "value": values}, nil
	}
	if *f.arrayString != "" {
		return map[string]any{"kind": "Array[String]", "value": parseStringArray(*f.arrayString)}, nil
	}
	if *f.arrayInt != "" {
		values, err := parseIntArray(*f.arrayInt)
		if err != nil {
			return nil, fmt.Errorf("%s --array-int %w", f.label, err)
		}
		return map[string]any{"kind": "Array[int]", "value": values}, nil
	}
	if *f.arrayFloat != "" {
		values, err := parseFloatArray(*f.arrayFloat)
		if err != nil {
			return nil, fmt.Errorf("%s --array-float %w", f.label, err)
		}
		return map[string]any{"kind": "Array[float]", "value": values}, nil
	}
	if *f.arrayBool != "" {
		values, err := parseBoolArray(*f.arrayBool)
		if err != nil {
			return nil, fmt.Errorf("%s --array-bool %w", f.label, err)
		}
		return map[string]any{"kind": "Array[bool]", "value": values}, nil
	}
	return map[string]any{"kind": "Resource", "value": *f.resource}, nil
}

func parseVectorArray(value string, want int, label string) ([][]float64, error) {
	groups := strings.Split(value, ";")
	out := make([][]float64, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		parsed, err := parseFloatList(group, want, label)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("must contain at least one vector")
	}
	return out, nil
}

func parseStringArray(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func parseIntArray(value string) ([]int, error) {
	parts := strings.Split(value, ";")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("component %q must be an integer: %w", part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseFloatArray(value string) ([]float64, error) {
	parts := strings.Split(value, ";")
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, fmt.Errorf("component %q must be a number: %w", part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseBoolArray(value string) ([]bool, error) {
	parts := strings.Split(value, ";")
	out := make([]bool, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseBool(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("component %q must be true or false: %w", part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseFloatList(value string, want int, label string) ([]float64, error) {
	parts := strings.Split(value, ",")
	if len(parts) != want {
		return nil, fmt.Errorf("must be %d comma-separated numbers", want)
	}
	out := make([]float64, 0, want)
	for _, part := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, fmt.Errorf("%s component %q must be a number: %w", label, part, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseNameJSONPairs(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, value := range values {
		name, jsonVal, ok := strings.Cut(value, "=")
		if !ok || name == "" || jsonVal == "" {
			return nil, fmt.Errorf("--prop must use name=JSON_VALUE")
		}
		var decoded any
		if err := json.Unmarshal([]byte(jsonVal), &decoded); err != nil {
			return nil, fmt.Errorf("--prop %s value must be typed JSON: %w", name, err)
		}
		out[name] = decoded
	}
	return out, nil
}

func parseNameRawJSONPairs(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, value := range values {
		name, jsonVal, ok := strings.Cut(value, "=")
		if !ok || name == "" || jsonVal == "" {
			return nil, fmt.Errorf("--param must use name=JSON_VALUE")
		}
		var decoded any
		if err := json.Unmarshal([]byte(jsonVal), &decoded); err != nil {
			return nil, fmt.Errorf("--param %s value must be JSON: %w", name, err)
		}
		out[name] = decoded
	}
	return out, nil
}

func parseNameResourcePairs(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, value := range values {
		name, resourcePath, ok := strings.Cut(value, "=")
		if !ok || name == "" || resourcePath == "" {
			return nil, fmt.Errorf("--texture-param must use name=res://path")
		}
		if !strings.HasPrefix(resourcePath, "res://") {
			return nil, fmt.Errorf("--texture-param resource must be a res:// path: %s", resourcePath)
		}
		out[name] = resourcePath
	}
	return out, nil
}
