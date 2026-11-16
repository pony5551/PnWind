package validation

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_trans "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
)

type ErrorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

func ValidateStruct(s interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	validate := validator.New()
	err := validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}
	return errors
}

type User struct {
	Account  string `json:"account" validate:"required" label:"用户"`
	PassWord string `json:"pass_word" validate:"required" label:"密码"`
}

func ValidateMap01() {
	//获取数据
	account := ""
	pass_word := ""
	user := &User{}
	user.Account = account
	user.PassWord = pass_word

	//验证
	zh_ch := zh.New()
	validate := validator.New()
	uni := ut.New(zh_ch)
	trans, _ := uni.GetTranslator("zh")

	//通过label标签返回自定义错误内容
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		label := field.Tag.Get("label")
		if label == "" {
			return field.Name
		}
		return label
	})

	//验证器注册翻译器
	zh_trans.RegisterDefaultTranslations(validate, trans)

	err := validate.Struct(user)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			//log.Errorln(err)
			s := strings.Replace(err.Translate(trans), " ", "", 1000)
			fmt.Println(s)
		}
		return
	}
}

func ValidateMap02() {
	//验证
	zh_ch := zh.New()
	validate := validator.New()
	uni := ut.New(zh_ch)
	trans, _ := uni.GetTranslator("zh")
	//验证器注册翻译器
	zh_trans.RegisterDefaultTranslations(validate, trans)

	data := map[string]interface{}{
		"name":  "Arshiya Kiani",
		"email": "zytel3301#gmail.com",
		"details": map[string]interface{}{
			"family_members": map[string]interface{}{
				"father_name": "Micheal",
				"mother_name": "Hannah",
			},
			"salary": "1000",
			"phones": []map[string]interface{}{
				{
					"number": "11-111-1111",
					"remark": "home",
				},
				{
					"number": "22-222-2222",
					"remark": "work",
				},
			},
		},
	}

	// Rules must be set as the structure as the data itself. If you want to dive into the
	// map, just declare its rules as a map
	rules := map[string]interface{}{
		"name":  "min=4,max=6",
		"email": "required,email",
		"details": map[string]interface{}{
			"family_members": map[string]interface{}{
				"father_name": "required,min=4,max=32",
				"mother_name": "required,min=4,max=32",
			},
			"salary": "number",
			"phones": map[string]interface{}{
				"number": "required,min=4,max=32",
				"remark": "required,min=1,max=4",
			},
		},
	}

	errs := validate.ValidateMap(data, rules)

	if len(errs) > 0 {
		//fmt.Println(errs)
		for v, err := range errs {
			fmt.Println(v, err)

			//fmt.Println(err.Translate(trans))
		}

	} else {
		fmt.Println("The user is valid")
	}

}

//github.com/go-playground/validator/v10
func processErr(data map[string]interface{}, err map[string]interface{}) string {
	if err == nil { //如果为nil 说明校验通过
		return ""
	}

	for _, v := range err {
		fmt.Println("v===========", v)
	}

	//invalid, ok := err.(*validator.InvalidValidationError) //如果是输入参数无效，则直接返回输入参数错误
	//if ok {
	//	return "输入参数错误：" + invalid.Error()
	//}
	//validationErrs := err.(validator.ValidationErrors) //断言是ValidationErrors
	//for _, validationErr := range validationErrs {
	//	fieldName := validationErr.Field()                    //获取是哪个字段不符合格式
	//	field, ok := reflect.TypeOf(u).FieldByName(fieldName) //通过反射获取filed
	//	if ok {
	//		errorInfo := field.Tag.Get("reg_error_info") //获取field对应的reg_error_info tag值
	//		return fieldName + ":" + errorInfo           //返回错误
	//	} else {
	//		return "缺失reg_error_info"
	//	}
	//}
	return ""
}

func ValidateMap03() {
	data := map[string]interface{}{
		"name":  "Arshiya Kiani",
		"email": "zytel3301#gmail.com",
		"details": map[string]interface{}{
			"family_members": map[string]interface{}{
				"father_name": "Micheal",
				"mother_name": "Hannah",
			},
			"salary": "1000",
			"phones": []map[string]interface{}{
				{
					"number": "11-111-1111",
					"remark": "home",
				},
				{
					"number": "22-222-2222",
					"remark": "work",
				},
			},
		},
	}

	// Rules must be set as the structure as the data itself. If you want to dive into the
	// map, just declare its rules as a map
	rules := map[string]interface{}{
		"name":  "min=4,max=6",
		"email": "required,email",
		"details": map[string]interface{}{
			"family_members": map[string]interface{}{
				"father_name": "required,min=4,max=32",
				"mother_name": "required,min=4,max=32",
			},
			"salary": "number",
			"phones": map[string]interface{}{
				"number": "required,min=4,max=32",
				"remark": "required,min=1,max=4",
			},
		},
	}

	validate := validator.New()
	errs := validate.ValidateMap(data, rules)

	if len(errs) > 0 {
		fmt.Println(errs)
		// The user is invalid
	} else {
		fmt.Println("The user is valid")
	}
}

func ValidateMap04() {
	var mapTemplate = map[string]interface{}{
		"name":       "required,alpha",
		"family":     "required,alpha",
		"email":      "required,email",
		"cell-phone": "numeric",
		"address": map[string]interface{}{
			"line1":       "required~line1为必填字段,alphanum",
			"line2":       "alphanum",
			"postal-code": "numeric",
		},
	}

	var inputMap = map[string]interface{}{
		"name":   "Bob",
		"family": "Smith",
		"email":  "foo@bar.baz",
		"address": map[string]interface{}{
			"line1":       "",
			"line2":       "",
			"postal-code": "",
		},
	}

	result, err := govalidator.ValidateMap(inputMap, mapTemplate)
	if err != nil {
		println("error: " + err.Error())
	}
	println(result)
}
