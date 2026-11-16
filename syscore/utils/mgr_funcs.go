package utils

import (
	"github.com/pony5551/PnWind/core/pnlibs"
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"github.com/pony5551/PnWind/sysmgr/models"

	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/pony5551/PnWind/core/validation/govalidator"

	"golang.org/x/exp/constraints"
)

const (
	Color_Red       = uint8(iota + 91)
	Color_Green     //	绿
	Color_Yellow    //	黄
	Color_Blue      // 	蓝
	Color_Magenta   //	洋红
	Color_LightBlue //浅蓝
	Color_Grey      //灰色
	Color_Black     //黑色
	Color_11
	Color_Black_BG_Grey
	Color_Black_BG_Red
	Color_Black_BG_Green
	Color_Black_BG_Yellow
	Color_Black_BG_Blue
	Color_Black_BG_Magenta
	Color_Black_BG_LightBlue
)

func ColorLog(s string, color uint8) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, s)
}

func IsNumber(s string) bool {
	// 去除首尾空格
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		// 存在 e 或 E, 判断是否为科学计数法
		if s[i] == 'e' || s[i] == 'E' {
			return IsSciNum(s[:i], s[i+1:])
		}
	}
	// 否则判断是否为整数或小数
	return IsInt(s) || IsDec(s)
}

// 是否为科学计数法
func IsSciNum(num1, num2 string) bool {
	// e 前后字符串长度为0 是错误的
	if len(num1) == 0 || len(num2) == 0 {
		return false
	}
	// e 后面必须是整数，前面可以是整数或小数  4  +
	return (IsInt(num1) || IsDec(num1)) && IsInt(num2)
}

// 判断是否为小数
func IsDec(s string) bool {
	// eg: 11.15, -0.15, +10.15, 3., .15,
	// err: +. 0..
	match1, _ := regexp.MatchString(`^[\+-]?\d*\.\d+$`, s)
	match2, _ := regexp.MatchString(`^[\+-]?\d+\.\d*$`, s)
	return match1 || match2
}

// 判断是否为整数
func IsInt(s string) bool {
	match, _ := regexp.MatchString(`^[\+-]?\d+$`, s)
	return match
}

func IsPositiveNumber(v any, _ any) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case []uint:
		for _, vv := range v.([]uint) {
			if !IsPositiveNumber(vv, nil) {
				return false
			}
		}
		return true
	case []int:
		for _, vv := range v.([]int) {
			if !IsPositiveNumber(vv, nil) {
				return false
			}
		}
		return true
	case int:
		break
	case int16:
		break
	case int32:
		break
	case int64:
		break
	case uint:
		break
	case uint16:
		break
	case uint32:
		break
	case uint64:
		break
	case float32:
		break
	case float64:
		break
	default:
		//logrus.Warnf("field is not type numeric: '%s'", v)
		return false
	}

	if i, ok := v.(uint); ok {
		return i > 0
	}

	if i, ok := v.(int); ok {
		return i > 0
	}

	if i, ok := v.(int64); ok {
		return i > 0
	}

	if i, ok := v.(int32); ok {
		return i > 0
	}

	if i, ok := v.(uint32); ok {
		return i > 0
	}

	if i, ok := v.(uint64); ok {
		return i > 0
	}

	if i, ok := v.(float64); ok {
		return i > 0
	}

	if i, ok := v.(float32); ok {
		return i > 0
	}

	//logrus.Errorf("unsupported type for positive check: %+v (%s)", v, reflect.TypeOf(v).Kind().String())
	return false
}

func IsSlice(v any) bool {
	return reflect.TypeOf(v).Kind() == reflect.Slice
}

func GetInterfaceToString(value any) string {
	// interface 转 string
	var key string
	if value == nil {
		return key
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}

	return key
}

func GMin[T constraints.Ordered](x, y T) T {
	return x + y
}

// email verify
func VerifyEmailFormat(email string) bool {
	//pattern := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*` //
	pattern := `^[0-9a-z][_.0-9a-z-]{0,31}@([0-9a-z][0-9a-z-]{0,30}[0-9a-z]\.){1,4}[a-z]{2,4}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// mobile verify
func VerifyMobileFormat(mobileNum string) bool {
	//regular := "^((13[0-9])|(14[5,7])|(15[0-3,5-9])|(17[0,3,5-8])|(18[0-9])|166|196|198|199|(147))\\d{8}$"
	regular := "^1[3456789]\\d{9}$"
	reg := regexp.MustCompile(regular)
	return reg.MatchString(mobileNum)
}

// telcom verify
func VerifyTelFormat(telNum string) bool {
	regular := "^(0[0-9]{2,3}\\-)?([2-9][0-9]{6,7})$"
	reg := regexp.MustCompile(regular)
	return reg.MatchString(telNum)
}

func IArrToString(arr any, tag string) string {
	result := ""
	//fmt.Println("type: ", reflect.TypeOf(arr).Name(), reflect.TypeOf(arr).Kind())
	if reflect.TypeOf(arr).Kind() != reflect.Slice {
		return result
	}

	LArr := arr.([]any)
	for _, i := range LArr { //遍历数组中所有元素追加成string
		s := i.(string)
		if result == "" {
			result = s
		} else {
			result = result + tag + s
		}
	}
	return result
}

func IArrToSArr(arr []any) []string {
	result := []string{}
	for _, i := range arr { //遍历数组中所有元素追加成string
		s := i.(string)
		result = append(result, s)
	}
	return result
}

func CheckIsNumberArray(params map[string]any, tag string) bool {
	if _, ok := params[tag]; !ok {
		return false
	}
	if reflect.TypeOf(params[tag]).Kind() != reflect.Slice {
		return false
	}
	ids := params[tag].([]any)
	if len(ids) < 1 {
		return false
	}
	idKind := reflect.TypeOf(ids[0]).Kind()
	ret := false
	switch idKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ret = true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ret = true
	case reflect.Float32, reflect.Float64:
		ret = true
	}
	return ret
}

func CheckIsStringArray(params map[string]any, tag string) bool {
	if _, ok := params[tag]; !ok {
		return false
	}
	if reflect.TypeOf(params[tag]).Kind() != reflect.Slice {
		return false
	}
	ids := params[tag].([]any)
	if len(ids) < 1 {
		return false
	}
	idKind := reflect.TypeOf(ids[0]).Kind()
	if idKind == reflect.String {
		return true
	}
	return false
}

func CheckIsMapArray(params any) bool {
	if reflect.TypeOf(params).Kind() != reflect.Slice {
		return false
	}
	ids := params.([]any)
	if len(ids) < 1 {
		return false
	}
	idKind := reflect.TypeOf(ids[0]).Kind()
	if idKind == reflect.Map {
		return true
	}
	return false
}

func GoValidatorMap(params map[string]any, validRulesMap map[string]any) (bool, error) {
	govalidator.CustomTypeTagMap.Set("numeric64", func(i any, o any) bool {
		//fmt.Println("numeric64===========: ", i, reflect.TypeOf(i))
		return reflect.TypeOf(i).Kind() == reflect.Float64
	})

	govalidator.CustomTypeTagMap.Set("IsSlice", func(i any, o any) bool {
		//fmt.Println("IsSlice===========1: ", reflect.TypeOf(i))
		if reflect.TypeOf(i).Kind() == reflect.Slice {
			return true
		}
		return false
	})

	result, err := govalidator.ValidateMap(params, validRulesMap)
	if err != nil {
		//fmt.Println("govalidator.error: " + err.Error())
		return false, err
	}
	//fmt.Println("govalidator: ", result)
	return result, err
}

func IdsSliceToStr(ids any) string {
	LIds := ids.([]any)
	var idArr []string
	for _, v := range LIds {
		idArr = append(idArr, strconv.FormatInt(int64(v.(float64)), 10))
	}
	return strings.Join(idArr, ",")
}

func IdsSliceToStrV2[T int64 | string](ids []T) string {
	s := ""
	for _, v := range ids {
		s += "'"
		s += fmt.Sprintf("%v", v)
		s += "'"
		s += ","
	}
	s1 := s[:len(s)-1]
	return s1
}

func IdsSliceToStrV3(ids any) []string {
	LIds := ids.([]any)
	var idArr []string
	for _, v := range LIds {
		idArr = append(idArr, strconv.FormatInt(int64(v.(float64)), 10))
	}
	return idArr
}

func CodesSliceToStr(ids any) string {
	LIds := ids.([]any)
	var idArr []string
	for _, v := range LIds {
		idArr = append(idArr, v.(string))
	}
	if len(idArr) < 1 {
		return ""
	}
	return "'" + strings.Join(idArr, "','") + "'"
}

// sysConfigLoad 让 LoadSysConfig 一次只跑一个：两次加载交错时，后读到的结果会被先读到的覆盖、删掉
var sysConfigLoad sync.Mutex

// LoadSysConfig 把参数表 db_sys_config 读进 objs（系统参数缓存）：键是 "分组.参数名"，值是参数值。
// 先把整张表读完再动 objs：中途出错时 objs 保持原样；表里已经删掉的参数，也从 objs 里去掉。
func LoadSysConfig(db *sqlx.DB, objs *pnlibs.PnObjects[string, string]) error {
	sysConfigLoad.Lock()
	defer sysConfigLoad.Unlock()

	confs := map[string]string{}
	if err := GetSysConfig(db, confs); err != nil {
		return err
	}
	for k, v := range confs {
		objs.Store(k, v)
	}
	// 表里已经没有的键：先收集再删，Range 里面不能调 Delete（它们要拿同一个分片的锁）
	var gone []string
	objs.Range(func(k string, _ string) bool {
		if _, isOk := confs[k]; !isOk {
			gone = append(gone, k)
		}
		return true
	})
	for _, k := range gone {
		objs.Delete(k)
	}
	return nil
}

// GetSysConfig 把参数表 db_sys_config 读进 confmaps：键是 "分组.参数名"，值是参数值。
func GetSysConfig(db *sqlx.DB, confmaps map[string]string) error {
	strSQL := "select * from db_sys_config order By sort,config_id asc"
	dbRows, err := db.Queryx(strSQL)
	if err != nil {
		return err
	}
	defer dbRows.Close()
	for dbRows.Next() {
		v := models.SysConfig{}
		err = dbRows.StructScan(&v) //dbRows.MapScan(mn),直接扫描到map值不受控制
		if err != nil {
			log.Errorln(err)
			return err
		}
		sKey := fmt.Sprintf("%s.%s", v.Sort, v.Key)
		confmaps[sKey] = v.Value.String
	}
	return dbRows.Err() // 读到一半断开时 Next 只是返回 false，错误在这里；不查的话半张表会被当成整张表
}

// JsonbParam 把一段 JSON 变成可以传给 jsonb 列的 SQL 参数：空的发 NULL，其余转成 string。
// []byte 类的值（[]byte、json.RawMessage、json.Marshal 的结果）传给 jsonb 列之前都要过它，本来就是 string 的不用：
// PostgreSQL 的连接串开着 binary_parameters（见 core/database），[]byte 会按二进制格式发给数据库，
// jsonb 列不认，报 unsupported jsonb version number。
func JsonbParam(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func DbInsert(db *sqlx.DB, tableName string, params map[string]any) (sql.Result, error) {
	keyStr := ""
	valueStr := ""
	split := ""
	for k, _ := range params {
		keyStr += split + k
		valueStr += split + ":" + k
		split = ","
	}
	strSQL := fmt.Sprintf("insert into %s(%s) values(%s)", tableName, keyStr, valueStr)
	return db.NamedExec(strSQL, params)
}

func DbInsertReturnID(db *sqlx.DB, tableName string, idField string, params map[string]any) (*sqlx.Rows, error) {
	keyStr := ""
	valueStr := ""
	split := ""
	for k, _ := range params {
		keyStr += split + k
		valueStr += split + ":" + k
		split = ","
	}
	strSQL := fmt.Sprintf("insert into %s(%s) values(%s) RETURNING %s", tableName, keyStr, valueStr, idField)
	return db.NamedQuery(strSQL, params)
}

func DbUpdate(db *sqlx.DB, tableName string, data map[string]any, condition map[string]any) (sql.Result, error) {
	keyStr := ""
	split := ""
	binding := make([]any, 0, len(data)+len(condition))
	p := 1
	for k, v := range data {
		keyStr += fmt.Sprintf("%s%s=$%d", split, k, p)
		binding = append(binding, v)
		split = ","
		p++
	}
	conditionStr := ""
	split = ""
	for k, v := range condition {
		conditionStr += fmt.Sprintf("%s%s$%d", split, k, p)
		split = " and "
		binding = append(binding, v)
		p++
	}
	strSQL := fmt.Sprintf("update %s SET %s WHERE %s", tableName, keyStr, conditionStr)
	return db.Exec(strSQL, binding...)
}

var ValidatorInstance = validator.New()

// ValidatorConvertErr 自定义validator包的错误信息
func ValidatorConvertErr(err error, msgMapping map[string]string) error {
	var validationErrs validator.ValidationErrors
	if ok := errors.As(err, &validationErrs); ok {
		for _, firstErr := range validationErrs {
			fieldName := firstErr.StructField()
			msgKey := fmt.Sprintf("%s.%s", fieldName, firstErr.Tag())
			if msg, ok := msgMapping[msgKey]; ok {
				return errors.New(msg)
			} else if msg, ok := msgMapping[fieldName]; ok {
				return errors.New(msg)
			}
		}
	}
	return err
}

func init() {
	govalidator.SetNilPtrAllowedByRequired(true)
}
