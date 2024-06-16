package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"

	"github.com/arklib/ark/errx"
)

type Validator struct {
	*validator.Validate

	UT          *ut.UniversalTranslator
	DefaultLang string
}

func New(lang string) *Validator {
	vd := validator.New()
	// set vd tag
	vd.SetTagName("vd")

	// set custom name tag
	vd.RegisterTagNameFunc(func(field reflect.StructField) string {
		label := field.Tag.Get("label")
		if label != "" {
			return label
		}
		return field.Name
	})

	// use universal
	enLocale := en.New()
	zhLocale := zh.New()
	uni := ut.New(enLocale,
		enLocale,
		zhLocale,
	)
	enTrans, _ := uni.GetTranslator("en")
	_ = enTranslations.RegisterDefaultTranslations(vd, enTrans)

	zhTrans, _ := uni.GetTranslator("zh")
	_ = zhTranslations.RegisterDefaultTranslations(vd, zhTrans)

	return &Validator{
		Validate:    vd,
		UT:          uni,
		DefaultLang: lang,
	}
}

func (v *Validator) Test(value any, lang string) error {
	err := v.Struct(value)
	if err != nil {
		locales := v.parseLocales(lang)
		trans, found := v.UT.FindTranslator(locales...)
		if !found {
			return err
		}

		err1 := err.(validator.ValidationErrors)[0]
		return errx.New(err1.Translate(trans))
	}
	return nil
}

func (v *Validator) parseLocales(lang string) []string {
	if lang == "" {
		return []string{v.DefaultLang}
	}

	var locales []string
	for _, value := range strings.Split(lang, ",") {
		locale := strings.Split(value, ";")
		locales = append(locales, locale[0])
	}
	// add default lang
	locales = append(locales, v.DefaultLang)
	return locales
}
