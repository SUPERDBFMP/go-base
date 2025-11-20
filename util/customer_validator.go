package util

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 自定义正则校验函数 - 身份证号
func idCardNoRegex(fl validator.FieldLevel) bool {
	reg := regexp.MustCompile(`^\d{6}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)
	return reg.MatchString(fl.Field().String())
}

// 自定义正则校验函数 - 手机号
func mobileRegex(fl validator.FieldLevel) bool {
	// 中国大陆手机号正则
	reg := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return reg.MatchString(fl.Field().String())
}

func carLicenseNoRegex(fl validator.FieldLevel) bool {
	// 中国大陆车牌号正则
	reg := regexp.MustCompile(`^[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤青藏川宁琼使领][A-HJ-NP-Z]([A-HJ-NP-Z0-9]{5}|[DF][A-HJ-NP-Z0-9]{5}|[A-HJ-NP-Z0-9]{5}[DF])$`)
	return reg.MatchString(fl.Field().String())
}

func timeRegex(fl validator.FieldLevel) bool {
	// 时间正则
	reg := regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01]) (0\d|1\d|2[0-3]):([0-5]\d):([0-5]\d)\.\d{3}$`)
	return reg.MatchString(fl.Field().String())
}

func InitValidator(validatorMap map[string]validator.Func) {
	// 注册自定义校验规则
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("idCardNoRegex", idCardNoRegex); err != nil {
			panic(err)
		}
		if err := v.RegisterValidation("mobileRegex", mobileRegex); err != nil {
			panic(err)
		}
		if err := v.RegisterValidation("carLicenseNoRegex", carLicenseNoRegex); err != nil {
			panic(err)
		}
		if err := v.RegisterValidation("timeRegex", timeRegex); err != nil {
			panic(err)
		}
		for key, value := range validatorMap {
			if err := v.RegisterValidation(key, value); err != nil {
				panic(err)
			}
		}
	}
}
