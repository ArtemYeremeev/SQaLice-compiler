package compiler

import (
	"regexp"
	"strconv"
	"strings"
)

var mathOperatorsList = []string{"==", "!=", "<=", "<", ">=", ">>", ">"}

// CondExpr describes structure of query condition
type CondExpr struct {
	FieldName string
	Operator  string
	Value     interface{}
	IsBracket bool
}

// GetFieldsList returns a list of query fields in SQL format
func GetFieldsList(fieldsMap map[string]string, q string) (fieldsList []string, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	fieldsBlock := strings.Split(q, "?")[0]
	if fieldsBlock == "" { // if fieldsBlock is empty then request all fields
		return nil, nil
	}
	jsonFields := strings.Split(fieldsBlock, ",")

	var sqlFields []string
	for _, f := range jsonFields {
		field := fieldsMap[f]
		if field == "" {
			return nil, newError("Passed unexpected field name in select - " + f)
		}

		sqlFields = append(sqlFields, field)
	}

	return sqlFields, nil
}

// GetConditionsList returns a list of all query conditions in SQL format
func GetConditionsList(fieldsMap map[string]string, q string) (condExprsList []*CondExpr, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	condsBlock := strings.Split(q, "?")[1]
	if condsBlock == "" { // if condsBlock is empty then request conds not passed
		return nil, nil
	}

	condsArray := regexp.MustCompile("[*|]").Split(condsBlock, -1) // split condsBlock by logicalOperators list
	var respArray []*CondExpr
	for _, cond := range condsArray {
		if cond == "" {
			continue
		}

		expr, err := extractQueryCondition(fieldsMap, cond)
		if err != nil {
			return nil, err
		}

		respArray = append(respArray, expr)
	}

	return respArray, nil
}

// GetConditionByName returns first condition with passed name and operator
func GetConditionByName(fieldsMap map[string]string, q string, fieldName string) (condExpr *CondExpr, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}
	if fieldName == "" {
		return nil, newError("Condition field name not passed")
	}

	condsBlock := strings.Split(q, "?")[1]
	if condsBlock == "" { // if condsBlock is empty then request conds not passed
		return nil, nil
	}

	condsArray := regexp.MustCompile("[*|]").Split(condsBlock, -1) // split condsBlock by logicalOperators list
	for _, cond := range condsArray {
		if cond == "" {
			continue
		}

		var operator string
		for _, o := range mathOperatorsList {
			if strings.Contains(cond, o) {
				operator = o
				break
			}
		}
		if operator == "" {
			return nil, newError("Unsupported operator in condition " + operator)
		}

		f := strings.Split(cond, operator)[0]
		if fieldName != f && "("+fieldName != f {
			continue
		}

		return extractQueryCondition(fieldsMap, cond)
	}

	return nil, nil
}

// GetSortField returns selection sort field from query
func GetSortField(fieldsMap map[string]string, q string) (field string, err error) {
	if q == "" {
		return "", newError("Query string not passed")
	}
	restsBlock := strings.Split(q, "?")[2]
	if restsBlock == "" { // if condsBlock is empty then sort field not passed
		return "", nil
	}
	f := strings.Split(restsBlock, ",")[0]
	if f == "" {
		return "", nil // if field is empty then sort field not passed
	}

	sortField := fieldsMap[f]
	if sortField == "" {
		return "", newError("Passed unexpected selection order field - " + f)
	}

	return sortField, nil
}

// GetSortOrder returns selection order from query
func GetSortOrder(q string) (order string, err error) {
	if q == "" {
		return "", newError("Query string not passed")
	}
	restsBlock := strings.Split(q, "?")[2]
	if restsBlock == "" { // if condsBlock is empty then sort order not passed
		return "", nil
	}

	sortOrder := strings.Split(restsBlock, ",")[1]
	if sortOrder == "" {
		return "", nil // if order is empty then sort order not passed
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		return "", newError("Passed unexpected selection order - " + sortOrder)
	}

	return sortOrder, nil
}

// GetLimit returns selection limit from query
func GetLimit(q string) (limit *int, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	restsBlock := strings.Split(q, "?")[2]
	if restsBlock == "" { // if condsBlock is empty then limit not passed
		return nil, nil
	}
	l := strings.Split(restsBlock, ",")[2]

	respLimit, err := strconv.Atoi(l)
	if err != nil {
		return nil, newError("Unexpected selection limit - " + l)
	}
	if respLimit < 0 {
		return nil, newError("Invalid negative selection limit - " + l)
	}

	return &respLimit, nil
}

// GetOffset returns selection offset from query
func GetOffset(q string) (limit *int, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	restsBlock := strings.Split(q, "?")[2]
	if restsBlock == "" { // if condsBlock is empty then offset not passed
		return nil, nil
	}
	o := strings.Split(restsBlock, ",")[3]

	respOffset, err := strconv.Atoi(o)
	if err != nil {
		return nil, newError("Unexpected selection offset - " + o)
	}
	if respOffset < 0 {
		return nil, newError("Invalid negative selection offset - " + o)
	}

	return &respOffset, nil
}

func extractQueryCondition(fieldsMap map[string]string, cond string) (condExpr *CondExpr, err error) {
	var op string // get condition operator
	for _, o := range mathOperatorsList {
		if strings.Contains(cond, o) {
			op = o
		}
		if strings.Contains(cond, o+"=") {
			op = o + "="
			break
		}
		if strings.Contains(cond, o+">") {
			op = o + ">"
			break
		}
	}
	if op == "" {
		return nil, newError("Unsupported operator in condition - " + cond)
	}

	isBracket := false // handle bracket condition
	bracketCond := regexp.MustCompile(`\((.*?)\)`).FindAllString(cond, -1)
	if bracketCond != nil {
		cond = strings.Trim(cond, "(")
		cond = strings.Trim(cond, ")")
		isBracket = true
	}

	f := strings.Split(cond, op)[0] // handle condition field name
	fieldName := fieldsMap[f]
	if fieldName == "" {
		return nil, newError("Passed unexpected field name in condition - " + f)
	}

	return &CondExpr{
		FieldName: fieldName,
		Operator:  operatorBindings[op],
		Value:     strings.Split(cond, op)[1],
		IsBracket: isBracket,
	}, nil
}
