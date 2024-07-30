package gensenc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

var ErrCantSet error = errors.New("cannot set")

func EncodeValue(v reflect.Value) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	l := make([]byte, 8)
	switch v.Type().Kind() {
	case reflect.String:
		val := v.String()
		binary.LittleEndian.PutUint64(l, uint64(len([]byte(val))))
		buf.Write(l)
		buf.Write([]byte(val))
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			b, err := EncodeValue(v.Field(i))
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
	case reflect.Slice:
		binary.LittleEndian.PutUint64(l, uint64(v.Len()))
		buf.Write(l)
		for i := 0; i < v.Len(); i++ {
			b, err := EncodeValue(v.Index(i))
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
	case reflect.Array:
		for i := range v.Len() {
			b, err := EncodeValue(v.Index(i))
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
	case reflect.Map:
		binary.LittleEndian.PutUint64(l, uint64(v.Len()))
		buf.Write(l)
		for _, key := range v.MapKeys() {
			b, err := EncodeValue(key)
			if err != nil {
				return nil, err
			}
			buf.Write(b)
			b, err = EncodeValue(v.MapIndex(key))
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		binary.LittleEndian.PutUint64(l, uint64(v.Int()))
		buf.Write(l)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		binary.LittleEndian.PutUint64(l, v.Uint())
		buf.Write(l)
	default:
		if v.CanInterface() {
			err := binary.Write(buf, binary.LittleEndian, v.Interface())
			if err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

func DecodeValue(r io.Reader, v reflect.Value) error {
	l := make([]byte, 8)
	switch v.Type().Kind() {
	case reflect.String:
		if !v.CanSet() {
			return ErrCantSet
		}
		_, err := io.ReadFull(r, l)
		if err != nil {
			return err
		}
		length := binary.LittleEndian.Uint64(l)
		b := make([]byte, length)
		_, err = io.ReadFull(r, b)
		if err != nil {
			return err
		}
		v.SetString(string(b))
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			err := DecodeValue(r, v.Field(i))
			if err != nil {
				return err
			}
		}
	case reflect.Slice:
		_, err := io.ReadFull(r, l)
		if err != nil {
			return err
		}
		length := binary.LittleEndian.Uint64(l)
		v.Clear()
		v.Grow(int(length))
		v.SetLen(int(length))
		for i := 0; i < int(length); i++ {
			err = DecodeValue(r, v.Index(i))
			if err != nil {
				return err
			}
		}
	case reflect.Array:
		for i := range v.Len() {
			err := DecodeValue(r, v.Index(i))
			if err != nil {
				return err
			}
		}
	case reflect.Map:
		_, err := io.ReadFull(r, l)
		if err != nil {
			return err
		}
		length := binary.LittleEndian.Uint64(l)
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		v.Clear()
		for range length {
			key := reflect.New(v.Type().Key())
			err = DecodeValue(r, key)
			if err != nil {
				return err
			}
			value := reflect.New(v.Type().Elem())
			err = DecodeValue(r, value)
			if err != nil {
				return err
			}
			v.SetMapIndex(key.Elem(), value.Elem())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !v.CanSet() {
			return ErrCantSet
		}
		_, err := io.ReadFull(r, l)
		if err != nil {
			return err
		}
		v.SetInt(int64(binary.LittleEndian.Uint64(l)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !v.CanSet() {
			return ErrCantSet
		}
		_, err := io.ReadFull(r, l)
		if err != nil {
			return err
		}
		v.SetUint(binary.LittleEndian.Uint64(l))
	case reflect.Pointer:
		err := DecodeValue(r, v.Elem())
		if err != nil {
			return err
		}
	default:
		if !v.CanSet() {
			return ErrCantSet
		}
		if v.CanInterface() {
			inter := v.Interface()
			err := binary.Read(r, binary.LittleEndian, inter)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(inter))
		}
	}
	return nil
}

func Encode(a any) ([]byte, error) {
	v := reflect.ValueOf(a)
	return EncodeValue(v)
}

func Decode(b []byte, a any) error {
	v := reflect.ValueOf(a)
	r := bytes.NewReader(b)
	return DecodeValue(r, v)
}
