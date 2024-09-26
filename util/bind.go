package util

import (
	"fmt"
	"reflect"

	"github.com/spf13/cast"
)

// BindStructFromMap 从 map 中根据 ctx 标签绑定值到结构体
func BindStructFromMap(input any, tagName string, data map[string]any) error {
	valueOf := reflect.ValueOf(input).Elem()
	typeOf := valueOf.Type()

	for i := 0; i < valueOf.NumField(); i++ {
		fieldValue := valueOf.Field(i)
		fieldType := typeOf.Field(i)

		key := fieldType.Tag.Get(tagName)
		if key == "" {
			continue
		}

		value, ok := data[key]
		if !ok {
			return fmt.Errorf("cannot set value for field %s", fieldType.Name)
		}

		err := setFieldValue(fieldValue, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// setFieldValue 根据值和字段类型设置结构体字段的值
func setFieldValue(fieldValue reflect.Value, value any) error {
	switch fieldValue.Kind() {
	case reflect.String:
		stringValue, err := cast.ToStringE(value)
		if err != nil {
			return fmt.Errorf("invalid value type for string")
		}
		fieldValue.SetString(stringValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		int64Value, err := cast.ToInt64E(value)
		if err != nil {
			return fmt.Errorf("invalid value type for int")
		}
		fieldValue.SetInt(int64Value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uint64Value, err := cast.ToUint64E(value)
		if err != nil {
			return fmt.Errorf("invalid value type for uint")
		}
		fieldValue.SetUint(uint64Value)
	case reflect.Slice:
		return setSliceFieldValue(fieldValue, value)
	default:
		return fmt.Errorf("unsupported field type")
	}
	return nil
}

// setSliceFieldValue 设置切片类型的字段值
func setSliceFieldValue(fieldValue reflect.Value, value any) error {
	switch fieldValue.Type().Elem().Kind() {
	case reflect.String:
		stringSlice, err := cast.ToStringSliceE(value)
		if err != nil {
			return fmt.Errorf("invalid value type for []string")
		}
		sliceValue := reflect.MakeSlice(reflect.TypeOf([]string{}), len(stringSlice), len(stringSlice))
		for i, v := range stringSlice {
			sliceValue.Index(i).SetString(v)
		}
		fieldValue.Set(sliceValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intSlice, err := cast.ToIntSliceE(value)
		if err != nil {
			return fmt.Errorf("invalid value type for []int")
		}
		sliceValue := reflect.MakeSlice(reflect.TypeOf([]int{}), len(intSlice), len(intSlice))
		for i, v := range intSlice {
			sliceValue.Index(i).SetInt(int64(v))
		}
		fieldValue.Set(sliceValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintSlice, err := cast.ToIntSliceE(value)
		if err != nil {
			return fmt.Errorf("invalid value type for []uint")
		}
		sliceValue := reflect.MakeSlice(reflect.TypeOf([]int{}), len(uintSlice), len(uintSlice))
		for i, v := range uintSlice {
			sliceValue.Index(i).SetUint(uint64(v))
		}
		fieldValue.Set(sliceValue)
	default:
		return fmt.Errorf("unsupported slice type")
	}
	return nil
}
