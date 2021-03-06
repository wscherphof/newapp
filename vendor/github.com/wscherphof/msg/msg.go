/*
Package msg manages translations of text labels ("messages") in a web application.

New messages are defined like this:
	msg.Key("Hello").
	  Set("en", "Hello, world").
	  Set("nl", "Hallo wereld")
	msg.Key("Hi").
	  Set("en", "Hi").
	  Set("nl", "Hoi")

The user's language is determined from the "Accept-Language" request header.
Pass the http.Request pointer to Translator():
	t := msg.Translator(r)
Then get the translation:
	message := t.Get("Hi")

Environment variables:

MSG_DEFAULT: determines the default language to use, if no translation is found
matching the Accept-Language header. The default value for MSG_DEFAULT is "en".

GO_ENV: if not set to "production", then translations that resorted to the
default language get prepended with "D-", and failed translations, falling back
to the message key, get prepended with "X-".

Messages and Translators are stored in memory. Translators are cached on their
Accept-Language header value.
*/
package msg

import (
	"github.com/wscherphof/env"
	"net/http"
	"os"
	"strings"
)

var (
	production      = (env.Get("GO_ENV", "") == "production")
	defaultLanguage = &languageType{}
)

func init() {
	defaultLanguage.parse(env.Get("MSG_DEFAULT", "en"))
}

/*
MessageType hold the translations for a message.
*/
type MessageType map[string]string

/*
Set stores the translation of the message for the given language. Any old
value is overwritten.
*/
func (m MessageType) Set(language, translation string) MessageType {
	language = strings.ToLower(language)
	m[language] = translation
	return m
}

var messageStore = make(map[string]MessageType, 500)

/*
NumLang sets the initial capacity for translations in a new message.
*/
var NumLang = 10

/*
Key returns the message stored under the given key, if it doesn't exist yet,
it gets created.
*/
func Key(key string) (message MessageType) {
	if m, ok := messageStore[key]; ok {
		message = m
	} else {
		message = make(MessageType, NumLang)
		messageStore[key] = message
	}
	return
}

type languageType struct {
	// e.g. "en-gb"
	Full string
	// e.g. "en"
	Main string
	// e.g. "gb"
	Sub string
}

func (l *languageType) parse(s string) {
	parts := strings.Split(s, "-")
	l.Full = s
	l.Main = parts[0]
	if len(parts) > 1 {
		l.Sub = parts[1]
	}
	return
}

var translatorCache = make(map[string]*TranslatorType, 100)

/*
TranslatorType knows about translations for the user's accepted languages.
*/
type TranslatorType struct {
	languages []*languageType
	files     map[string]string
}

/*
Translator returns a (cached) TranslatorType.
*/
func Translator(r *http.Request) *TranslatorType {
	acceptLanguage := strings.ToLower(r.Header.Get("Accept-Language"))
	if cached, ok := translatorCache[acceptLanguage]; ok {
		return cached
	}
	langStrings := strings.Split(acceptLanguage, ",")
	t := &TranslatorType{
		languages: make([]*languageType, len(langStrings)),
		files:     make(map[string]string, 20),
	}
	for i, v := range langStrings {
		langString := strings.Split(v, ";")[0] // cut the q parameter
		lang := &languageType{}
		lang.parse(langString)
		t.languages[i] = lang
	}
	translatorCache[acceptLanguage] = t
	return t
}

/*
Get returns the translation for a message.
*/
func (t *TranslatorType) Get(key string) (translation string) {
	if key == "" {
		return ""
	}
	for _, language := range t.languages {
		if translation = translate(key, language); translation != "" {
			return
		}
	}
	if translation = translate(key, defaultLanguage); translation != "" {
		if !production {
			translation = "D-" + translation
		}
	} else {
		translation = key
		if !production {
			translation = "X-" + translation
		}
	}
	return
}

func translate(key string, language *languageType) (translation string) {
	if val, ok := messageStore[key][language.Full]; ok {
		translation = val
	} else if val, ok := messageStore[key][language.Main]; ok {
		translation = val
	} else if val, ok := messageStore[key][language.Sub]; ok {
		translation = val
	}
	return
}

/*
File searches for an "inner" template fitting the "base" template, matching
the user's accepted languages.

Template names are without file name extension. The default extension is ".ace".

Example: if MSG_DEFAULT is "en", and the Accept-Languages header is empty,
	msg.File("/resources/templates", "home", "HomePage", ".tpl")
returns
	"HomePage-en", nil
if the file "/resources/templates/home/HomePage-en.tpl" exists.
*/
func (t *TranslatorType) File(location, dir, base string, extension ...string) (inner string, err error) {
	ext := ".ace"
	if len(extension) == 1 {
		ext = extension[0]
	}
	template := location + "/" + dir + "/" + base
	if cached, ok := t.files[template]; ok {
		return cached, nil
	}
	for _, language := range t.languages {
		if lang, e := exists(template, ext, language); e == nil {
			inner = base + "-" + lang
			t.files[template] = inner
			return
		}
	}
	if lang, e := exists(template, ext, defaultLanguage); e != nil {
		err = e
	} else {
		inner = base + "-" + lang
		t.files[template] = inner
	}
	return
}

func exists(template, extension string, language *languageType) (lang string, err error) {
	if lang, err = stat(template, extension, language.Full); err != nil {
		if lang, err = stat(template, extension, language.Main); err != nil {
			lang, err = stat(template, extension, language.Sub)
		}
	}
	return
}

func stat(template, extension, searchLang string) (lang string, err error) {
	path := template + "-" + searchLang + extension
	if _, err = os.Stat(path); err == nil {
		lang = searchLang
	}
	return
}
