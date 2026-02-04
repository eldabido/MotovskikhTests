package entities

const (
	LangRu = "ru"
	LangEn = "en"
)

// LangPrefixes отражают главную суть сайта с одним доменом —
// для английской версии используется папка en.
var LangPrefixes = map[string]string{
	LangEn: "en/",
	LangRu: "",
}
