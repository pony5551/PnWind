package utils

import (
	"github.com/pony5551/PnWind/core/pnlibs"
	"database/sql"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"reflect"
	"time"
)

// ========== 公开接口 ==========

// Default 包级别默认实例，直接使用 DbStructToMap 函数时走此实例。
var Default = NewMapper()

// DbStructToMap 将结构体按指定 tag 转换并写入 map。
// tag: 要读取的 struct tag 名称，如 "db"、"json"。
// v 可以是结构体或结构体指针。
func DbStructToMap(out map[string]interface{}, v interface{}, tag string) {
	Default.DbStructToMap(out, v, tag)
}

// Mapper 持有反射缓存，默认使用包级别的 Default 实例即可。
// 仅在需要缓存隔离时（如单元测试）才手动创建。
type Mapper struct {
	cache pnlibs.PnObjects[cacheKey, *structMeta]
}

func NewMapper() *Mapper {
	return &Mapper{
		cache: *pnlibs.NewPnObjects[cacheKey, *structMeta](cacheKeyHash),
	}
}

func (m *Mapper) DbStructToMap(out map[string]interface{}, v interface{}, tag string) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	meta := m.parseMeta(rv.Type(), tag)
	for _, f := range meta.fields {
		out[f.tagName] = extractValue(rv.Field(f.index), f.kind, f.fieldType)
	}
}

// Reset 清空缓存，主要用于测试。
func (m *Mapper) Reset() {
	m.cache.Clear(nil)
}

// ========== 内部实现 ==========

type cacheKey struct {
	rt  reflect.Type
	tag string
}

type structMeta struct {
	fields []fieldMeta
}

type fieldMeta struct {
	index     int
	tagName   string
	kind      reflect.Kind
	fieldType reflect.Type
}

var (
	typeNullString  = reflect.TypeOf(sql.NullString{})
	typeNullInt32   = reflect.TypeOf(sql.NullInt32{})
	typeNullInt64   = reflect.TypeOf(sql.NullInt64{})
	typeNullFloat64 = reflect.TypeOf(sql.NullFloat64{})
	typeNullBool    = reflect.TypeOf(sql.NullBool{})
	typeNullTime    = reflect.TypeOf(sql.NullTime{})
	typeTime        = reflect.TypeOf(time.Time{})
)

func cacheKeyHash(k cacheKey) uint64 {
	h := fnv.New64a()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(reflect.ValueOf(k.rt).Pointer()))
	h.Write(b)
	h.Write([]byte(k.tag))
	return h.Sum64()
}

func (m *Mapper) parseMeta(rt reflect.Type, tag string) *structMeta {
	key := cacheKey{rt, tag}
	if meta, ok := m.cache.Load(key); ok {
		return meta
	}
	meta := buildMeta(rt, tag)
	m.cache.LoadOrStore(key, meta)
	return meta
}

func buildMeta(rt reflect.Type, tag string) *structMeta {
	meta := &structMeta{}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		tagVal := f.Tag.Get(tag)
		if tagVal == "" || tagVal == "-" {
			continue
		}
		if !f.IsExported() {
			continue
		}
		meta.fields = append(meta.fields, fieldMeta{
			index:     i,
			tagName:   tagVal,
			kind:      f.Type.Kind(),
			fieldType: f.Type,
		})
	}
	return meta
}

func extractValue(fv reflect.Value, kind reflect.Kind, ft reflect.Type) interface{} {
	switch kind {
	case reflect.Bool:
		return fv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fv.Uint()
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.6f", fv.Float())
	case reflect.String:
		return fv.String()
	case reflect.Array, reflect.Slice, reflect.Map:
		return fv.Interface()
	case reflect.Struct:
		return extractStruct(fv, ft)
	default:
		return fv.Interface()
	}
}

func extractStruct(fv reflect.Value, ft reflect.Type) interface{} {
	switch ft {
	case typeNullString:
		v := fv.Interface().(sql.NullString)
		if !v.Valid {
			return nil
		}
		return v.String
	case typeNullInt32:
		v := fv.Interface().(sql.NullInt32)
		if !v.Valid {
			return nil
		}
		return v.Int32
	case typeNullInt64:
		v := fv.Interface().(sql.NullInt64)
		if !v.Valid {
			return nil
		}
		return v.Int64
	case typeNullFloat64:
		v := fv.Interface().(sql.NullFloat64)
		if !v.Valid {
			return nil
		}
		return v.Float64
	case typeNullBool:
		v := fv.Interface().(sql.NullBool)
		if !v.Valid {
			return nil
		}
		return v.Bool
	case typeNullTime:
		v := fv.Interface().(sql.NullTime)
		if !v.Valid {
			return nil
		}
		return v.Time
	case typeTime:
		return fv.Interface().(time.Time)
	default:
		return fv.Interface()
	}
}
