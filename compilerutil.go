package compiler

import (
	"errors"
	"reflect"
	"strings"
	"sort"
)

// FormDinamicModel forms a model containing fields for building query
func FormDinamicModel(model reflect.Value) map[string]string {
	modelTypes := model.Type()

	fieldsMap := make(map[string]string, model.NumField())
	for i := 0; i < model.NumField(); i++ { // json tag: sql tag
		fieldsMap[strings.TrimSuffix(modelTypes.Field(i).Tag.Get("json"), ",omitempty")] = modelTypes.Field(i).Tag.Get("sql")
	}

	return fieldsMap
}

func addPGQuotes(str string) string {
	return "'" + str + "'"
}

func newError(errText string) error {
	if errText == "" {
		errText = "Unexpected error"
	}
	return errors.New("[SQaLice] " + errText)
}

func sortMap(m map[string]string) []string {
	var keys []string
	for k := range m {
    	keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
