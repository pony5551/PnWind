package util

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	log "github.com/pony5551/PnWind/core/pnlogrus"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/valyala/bytebufferpool"
)

// 去重与去空
func RemoveRepeatElement(list []string) []string {
	// 创建一个临时map用来存储数组元素
	temp := make(map[string]bool)
	newList := []string{}
	for _, v := range list {
		if "" == strings.TrimSpace(v) {
			continue
		}
		// 遍历数组元素，判断此元素是否已经存在map中
		_, ok := temp[v]
		if !ok {
			temp[v] = true
			newList = append(newList, v)
		}
	}
	return newList
}

// 任意位置插入元素
func SliceInsertElement[T any](slice []T, item T, index int) []T {
	slice = append(slice[:index], append([]T{item}, slice[index:]...)...)
	return slice
}

func ToJSON(obj any) (string, error) {
	res, err := json.Marshal(obj)
	if err != nil {
		res = []byte("")
	}
	return string(res), err
}

func JsonToMap(str string) (map[string]any, error) {
	var tempMap map[string]any
	err := json.Unmarshal([]byte(str), &tempMap)
	if err != nil {
		//panic(err)
		log.Errorln("JsonToMap json.Unmarshal err:", err)
		return nil, err
	}
	return tempMap, nil
}

// 将结构体转换为map[string]any
func StructToMap(obj any, tagName string) map[string]any {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var data = make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		sTag := field.Tag.Get(tagName)
		data[sTag] = value.Interface()
	}
	return data
}

// MapToStruct 将map[string]any映射到结构体
func MapToStruct(m map[string]any, s any) error {
	structValue := reflect.ValueOf(s)
	if structValue.Kind() != reflect.Ptr || structValue.IsNil() {
		return fmt.Errorf("MapToStruct requires a pointer to a struct")
	}

	structValue = structValue.Elem()
	if structValue.Kind() != reflect.Struct {
		return fmt.Errorf("MapToStruct requires a pointer to a struct")
	}

	structType := structValue.Type()
	for i := 0; i < structValue.NumField(); i++ {
		structField := structType.Field(i)
		structFieldValue := structValue.Field(i)

		// 获取结构体字段的json标签作为映射键
		jsonTag := structField.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = strings.ToLower(structField.Name)
		}

		// 从map中获取对应键的值
		if value, ok := m[jsonTag]; ok {
			mapVal := reflect.ValueOf(value)
			if mapVal.Type().ConvertibleTo(structFieldValue.Type()) {
				structFieldValue.Set(mapVal.Convert(structFieldValue.Type()))
			}
		}
	}
	return nil
}

// GZ压缩
func GZipBytes(data []byte) []byte {
	var in bytes.Buffer
	g := gzip.NewWriter(&in) //面向api编程调用压缩算法的一个api
	//参数就是指向某个数据缓冲区默认压缩等级是DefaultCompression 在这里还有另一个api可以调用调整压缩级别
	//gzip.NewWirterLevel(&in,gzip.BestCompression) NoCompression（对应的int 0）、
	//BestSpeed（1）、DefaultCompression（-1）、HuffmanOnly（-2）BestCompression（9）这几个级别也可以
	//这样写gzip.NewWirterLevel(&in,0)
	//这里的异常返回最好还是处理下，我这里纯属省事
	g.Write(data)
	g.Close()
	return in.Bytes()
}

// GZ解压
func UnGZipBytes(data []byte) []byte {
	var out bytes.Buffer
	var in bytes.Buffer
	in.Write(data)
	r, _ := gzip.NewReader(&in)
	r.Close() //这句放在后面也没有问题，不写也没有任何报错
	//机翻注释：关闭关闭读者。它不会关闭底层的io.Reader。为了验证GZIP校验和，读取器必须完全使用，直到io.EOF。

	io.Copy(&out, r) //这里我看了下源码不是太明白，
	//我个人想法是这样的，Reader本身就是go中表示一个压缩文件的形式，r转化为[]byte就是一个符合压缩文件协议的压缩文件

	return out.Bytes()
}

// 返回一个32位md5加密后的字符串
func Md5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// Decimal float64 保留小数点后位数
// value float64 浮点数
// prec int 需保留小数点后的位数
func Decimal(value float64, prec int) float64 {
	value, _ = strconv.ParseFloat(strconv.FormatFloat(value, 'f', prec, 64), 64)
	return value
}

// 按字段排序
// sort.Slice(s, func(i, j int) bool { return lessCaseInsensitive(s[i], s[j]) })
func LessCaseInsensitive(s, t string) bool {
	for {
		if len(t) == 0 {
			return false
		}
		if len(s) == 0 {
			return true
		}
		c, sizec := utf8.DecodeRuneInString(s)
		d, sized := utf8.DecodeRuneInString(t)

		lowerc := unicode.ToLower(c)
		lowerd := unicode.ToLower(d)

		if lowerc < lowerd {
			return true
		}
		if lowerc > lowerd {
			return false
		}

		s = s[sizec:]
		t = t[sized:]
	}
}

// #nosec G103
// UnsafeString returns a string pointer without allocation
func UnsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func ParseParamSquareBrackets(k string) (string, error) {
	bb := bytebufferpool.Get()
	defer bytebufferpool.Put(bb)
	kbytes := []byte(k)

	for i, b := range kbytes {

		if b == '[' && kbytes[i+1] != ']' {
			if err := bb.WriteByte('.'); err != nil {
				return "", err
			}
		}

		if b == '[' || b == ']' {
			continue
		}

		if err := bb.WriteByte(b); err != nil {
			return "", err
		}
	}

	return bb.String(), nil
}

func ParseVendorSpecificContentType(cType string) string {
	plusIndex := strings.Index(cType, "+")

	if plusIndex == -1 {
		return cType
	}

	var parsableType string
	if semiColonIndex := strings.Index(cType, ";"); semiColonIndex == -1 {
		parsableType = cType[plusIndex+1:]
	} else if plusIndex < semiColonIndex {
		parsableType = cType[plusIndex+1 : semiColonIndex]
	} else {
		return cType[:semiColonIndex]
	}

	slashIndex := strings.Index(cType, "/")

	if slashIndex == -1 {
		return cType
	}

	return cType[0:slashIndex+1] + parsableType
}

func GetProcessName() string {
	pid := os.Getpid()
	pn, err := process.NewProcess(int32(pid))
	if err != nil {
		return ""
	}
	pName, err := pn.Name()
	if err != nil {
		return ""
	}
	return pName
}

func GetHttpServerByUrl(AUrl string) string {
	reg := regexp.MustCompile(`(http|https)\:\/\/[^/\s]+`)
	if reg == nil {
		return ""
	}

	result := reg.FindAllStringSubmatch(AUrl, -1)
	if len(result) < 1 || len(result[0]) < 1 {
		return ""
	}

	return result[0][0]
}

// 获取项目根目录
func getRootPath() string {

	// 第一种方式：获取当前执行程序所在的绝对路径
	// 这种仅在 `go build` 时，才可以获取正确的路径
	// 获取当前执行的二进制文件的全路径，包括二进制文件名
	// eg: exePath = "/var/folders/hr/2rqppbcx4kv8_3qc_ky1qcy80000gn/T/go-build265586886/b001/exe/main"
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// eg: rootPathByExecutable = "/private/var/folders/hr/2rqppbcx4kv8_3qc_ky1qcy80000gn/T/go-build265586886/b001/exe"
	rootPathByExecutable, _ := filepath.EvalSymlinks(filepath.Dir(exePath))

	// 第二种方式：获取当前执行文件绝对路径
	// 这种方式在 `go run` 和 `go build` 时，都可以获取到正确的路径
	// 但是交叉编译后，执行的结果是错误的结果
	var rootPathByCaller string
	// eg: filename = "/path/to/project/main.go"
	_, filename, _, ok := runtime.Caller(0)
	// eg: rootPathByCaller = "/path/to/project"
	if ok {
		rootPathByCaller = path.Dir(filename)
	}

	// 可以通过 `echo $TMPDIR` 查看当前系统临时目录
	// eg: tmpDir = "/private/var/folders/hr/2rqppbcx4kv8_3qc_ky1qcy80000gn/T"
	tmpDir, _ := filepath.EvalSymlinks(os.TempDir())

	// 对比通过 `os.Executable()` 获取到的路径是否与 `TMPDIR` 环境变量设置的路径相同
	// 相同，则说明是通过 `go run` 命令启动的
	// 不同，则是通过 `go build` 命令启动的
	if strings.Contains(rootPathByExecutable, tmpDir) {
		return rootPathByCaller
	}

	return rootPathByExecutable
}

// FormatNumbers 格式化用逗号分隔的数字字符串为用单引号包围且逗号分隔的字符串
func FormatNumbers(input string) (string, error) {
	// 将输入字符串分割成切片
	numbers := strings.Split(input, ",")

	// 正则表达式匹配数字
	numberPattern := regexp.MustCompile(`^\d+$`)

	// 用单引号包围每个数字并验证
	for i, num := range numbers {
		if numberPattern.MatchString(num) {
			numbers[i] = fmt.Sprintf("'%s'", num)
		} else {
			return "", errors.New(fmt.Sprintf("'%s' is not a valid number", num))
		}
	}

	// 将切片重新连接成一个字符串
	output := strings.Join(numbers, ",")

	return output, nil
}
