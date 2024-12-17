package translation

import (
	"github.com/leonelquinteros/gotext"
)

func GetLanguage() string {
	lang := gotext.GetLanguage()

	if lang == "und" || lang == "" {
		return "en"
	}

	return lang
}

func Translate(msgID string, vars ...interface{}) string {
	return gotext.Get(msgID, vars...)
}
