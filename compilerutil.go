package compiler

import (
	"errors"
	"reflect"
	"sort"
	"strings"
)

// formDinamicModel forms a model containing fields for building query
func formDinamicModel(model interface{}) map[string]string {
	reflectModel := reflect.ValueOf(model)
	modelTypes := reflectModel.Type()

	fieldsMap := make(map[string]string, reflectModel.NumField())
	for i := 0; i < reflectModel.NumField(); i++ { // json tag: sql tag
		if modelTypes.Field(i).Type.Kind() == reflect.Struct { // handle nested struct
			nestedFields := formDinamicModel(reflectModel.Field(i).Interface())
			for k, v := range nestedFields { // merge nestedFields into main map
				fieldsMap[k] = v
			}
			continue
		}

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

// sortMap sorts map elements in alphabetic order
func sortMap(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
