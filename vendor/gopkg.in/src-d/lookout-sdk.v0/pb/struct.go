package pb

import (
	"fmt"
	"reflect"

	types "github.com/gogo/protobuf/types"
)

// ToStruct converts a map[string]interface{} to a types.Struct
func ToStruct(v map[string]interface{}) *types.Struct {
	size := len(v)
	if size == 0 {
		return nil
	}

	fields := make(map[string]*types.Value, size)
	for k, v := range v {
		fields[k] = ToValue(v)
	}

	return &types.Struct{
		Fields: fields,
	}
}

func newBoolValue(v bool) *types.Value {
	return &types.Value{
		Kind: &types.Value_BoolValue{
			BoolValue: v,
		},
	}
}

func newNumberValue(v float64) *types.Value {
	return &types.Value{
		Kind: &types.Value_NumberValue{
			NumberValue: v,
		},
	}
}

// ToValue converts an interface{} to a types.Value
func ToValue(v interface{}) *types.Value {
	switch v := v.(type) {
	case nil:
		return nil
	case bool:
		return newBoolValue(v)
	case int:
		return newNumberValue(float64(v))
	case int8:
		return newNumberValue(float64(v))
	case int32:
		return newNumberValue(float64(v))
	case int64:
		return newNumberValue(float64(v))
	case uint:
		return newNumberValue(float64(v))
	case uint8:
		return newNumberValue(float64(v))
	case uint32:
		return newNumberValue(float64(v))
	case uint64:
		return newNumberValue(float64(v))
	case float32:
		return newNumberValue(float64(v))
	case float64:
		return newNumberValue(float64(v))
	case string:
		return &types.Value{
			Kind: &types.Value_StringValue{
				StringValue: v,
			},
		}
	default:
		return toValue(reflect.ValueOf(v))
	}
}

func toValue(v reflect.Value) *types.Value {
	switch v.Kind() {
	case reflect.Bool:
		return newBoolValue(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return newNumberValue(float64(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return newNumberValue(float64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return newNumberValue(v.Float())
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return toValue(reflect.Indirect(v))
	case reflect.Array, reflect.Slice:
		size := v.Len()
		if size == 0 {
			return nil
		}
		values := make([]*types.Value, size)
		for i := 0; i < size; i++ {
			values[i] = toValue(v.Index(i))
		}
		return &types.Value{
			Kind: &types.Value_ListValue{
				ListValue: &types.ListValue{
					Values: values,
				},
			},
		}
	case reflect.Struct:
		t := v.Type()
		size := v.NumField()
		if size == 0 {
			return nil
		}
		fields := make(map[string]*types.Value, size)
		for i := 0; i < size; i++ {
			name := t.Field(i).Name
			if len(name) > 0 {
				fields[name] = toValue(v.Field(i))
			}
		}
		if len(fields) == 0 {
			return nil
		}
		return &types.Value{
			Kind: &types.Value_StructValue{
				StructValue: &types.Struct{
					Fields: fields,
				},
			},
		}
	case reflect.Map:
		keys := v.MapKeys()
		if len(keys) == 0 {
			return nil
		}
		fields := make(map[string]*types.Value, len(keys))
		for _, k := range keys {
			if k.Kind() == reflect.String {
				fields[k.String()] = toValue(v.MapIndex(k))
			} else if k.Kind() == reflect.Interface {
				ik := k.Interface()
				sk, ok := ik.(string)
				if ok {
					fields[sk] = toValue(v.MapIndex(k))
				}
			}
		}
		if len(fields) == 0 {
			return nil
		}
		return &types.Value{
			Kind: &types.Value_StructValue{
				StructValue: &types.Struct{
					Fields: fields,
				},
			},
		}
	case reflect.Interface:
		return ToValue(v.Interface())
	default:
		return &types.Value{
			Kind: &types.Value_StringValue{
				StringValue: fmt.Sprint(v),
			},
		}
	}
}
