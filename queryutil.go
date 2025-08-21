package compiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var mathOperatorsList = []string{"==", "!=", "<=", "<", ">=", ">>", ">", "!!"}

// CondExpr describes structure of query condition
type CondExpr struct {
	FieldName    string
	Operator     string
	Value        interface{}
	IsBracket    bool
	SepOperator  string
}

// GetFieldsList returns a list of query fields in SQL format
func GetFieldsList(model interface{}, q string) (fieldsList []string, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

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
func GetConditionsList(model interface{}, q string, toDBFormat bool, isSearch bool) (condExprsList []*CondExpr, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	// form fields map with formDinamicModel, if its necessary
	var fieldsMap map[string]string
	if toDBFormat {
		fieldsMap = formDinamicModel(model)
	}

	// handle searchQuery conditions
	if isSearch {
		var respConds []*CondExpr
		for _, cond := range regexp.MustCompile("[*|]").Split(q, -1) {
			if cond == "" {
				continue
			}

			var condArr []string
			if condArr = strings.Split(cond, "~~"); len(condArr) < 2 {
				return nil, newError("Unsupported searchQuery format")
			}

			respConds = append(respConds, &CondExpr{FieldName: condArr[0], Operator:  "~~", Value: condArr[1]})
		}

		return respConds, nil
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

		expr, err := extractQueryCondition(fieldsMap, cond, toDBFormat)
		if err != nil {
			return nil, err
		}

		respArray = append(respArray, expr)
	}

	return respArray, nil
}

// GetConditionByName returns first condition with passed name and operator
func GetConditionByName(model interface{}, q string, fieldName string, toDBFormat bool) (condExpr *CondExpr, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}
	if fieldName == "" {
		return nil, newError("Condition field name not passed")
	}

	// form fields map with formDinamicModel, if its necessary
	var fieldsMap map[string]string
	if toDBFormat {
		fieldsMap = formDinamicModel(model)
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
			return nil, newError("Unsupported operator in condition")
		}

		f := strings.Split(cond, operator)[0]
		if fieldName != f && "("+fieldName != f {
			continue
		}

		return extractQueryCondition(fieldsMap, cond, toDBFormat)
	}

	return nil, nil
}

// GetSortField returns selection sort field from query
func GetSortField(model interface{}, q string) (fields []string, err error) {
	if q == "" {
		return nil, newError("Query string not passed")
	}

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

	restsBlock := strings.Split(q, "?")[2]
	if restsBlock == "" { // if condsBlock is empty then sort field not passed
		return nil, nil
	}
	flds := strings.Split(restsBlock, ",")[0]
	if flds == "" {
		return nil, nil // if field is empty then sort field not passed
	}

	var respFields []string
	for _, f := range strings.Split(flds, "|") {
		sortField := fieldsMap[f]
		if sortField == "" {
			return nil, newError("Passed unexpected selection order field - " + f)
		}

		respFields = append(respFields, sortField)
	}

	return respFields, nil
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
	if o == "" {
		o = "0"
	}

	respOffset, err := strconv.Atoi(o)
	if err != nil {
		return nil, newError("Unexpected selection offset - " + o)
	}
	if respOffset < 0 {
		return nil, newError("Invalid negative selection offset - " + o)
	}

	return &respOffset, nil
}

// AddQueryFieldsToSelect adds fieldsMap JSON or fieldsArray fields in SELECT block,
// replacing current fields in query if isDeleteCurrent passed as true
func AddQueryFieldsToSelect(model interface{}, query string, fieldsArray []string, isDeleteCurrent bool) (string, error) {
	if query == "" {
		return query, newError("Passed empty query for forming fields block")
	}
	queryBlocks := strings.Split(query, "?")

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

	var selectBlock []string
	// If fieldsMap passed, check if passed new fiedls correct
	if fieldsMap != nil {
		for _, key := range fieldsArray {
			// If field not found in fieldsMap, skip it
			if fieldsMap[key] != "" {
				selectBlock = append(selectBlock, key)
			}
		}
	} else {
		selectBlock = fieldsArray
	}

	if isDeleteCurrent {
		queryBlocks[0] = strings.Join(selectBlock, ",")
	} else {
		queryBlocks[0] = queryBlocks[0] + "," + strings.Join(selectBlock, ",")
	}

	return strings.Join(queryBlocks, "?"), nil
}

// AddQueryConditions adds conditions to query conditions list with AND separators
// If isDeleteCurrent argument passed, conds list peplacing current query conditions
// If correct sepOperator passed in condition, compiler use it instead of AND separator
func AddQueryConditions(query string, conds []CondExpr, isDeleteCurrent, isLeading bool) (string, error) {
	if query == "" {
		return query, newError("Passed empty query for forming conditions block")
	}
	if conds == nil {
		return query, nil
	}
	queryBlocks := strings.Split(query, "?")

	if isDeleteCurrent {
		queryBlocks[1] = ""
	}

	for _, cond := range conds {
		if cond.FieldName == "" {
			continue
		}

		isOperatorCorrect := false
		for _, key := range mathOperatorsList { // check condition operator
			if cond.Operator == key {
				isOperatorCorrect = true
			}
		}
		if !isOperatorCorrect {
			return query, newError("Passed incorrect operator in query condition - " + cond.Operator)
		}
		sepOperator := "*"
		if cond.SepOperator != "" && logicalBindings[sepOperator] != "" {
			sepOperator = cond.SepOperator
		}

		if isLeading {
			if queryBlocks[1] != "" { // separates conditions with AND logical operator
				queryBlocks[1] = sepOperator + queryBlocks[1]
			}
			if cond.IsBracket { // handle bracket condition
				queryBlocks[1] = "(" + cond.FieldName + cond.Operator + fmt.Sprintf("%v", cond.Value) + ")" + queryBlocks[1]
			} else {
				queryBlocks[1] = cond.FieldName + cond.Operator + fmt.Sprintf("%v", cond.Value) + queryBlocks[1]
			}

			continue
		}

		if queryBlocks[1] != "" { // separates conditions with AND logical operator
			queryBlocks[1] = queryBlocks[1] + sepOperator
		}
		if cond.IsBracket { // handle bracket condition
			queryBlocks[1] = queryBlocks[1] + "(" + cond.FieldName + cond.Operator + fmt.Sprintf("%v", cond.Value) + ")"
		} else {
			queryBlocks[1] = queryBlocks[1] + cond.FieldName + cond.Operator + fmt.Sprintf("%v", cond.Value)
		}
	}

	return strings.Join(queryBlocks, "?"), nil
}

// ReplaceQueryCondition replaces query condition by fieldName
func ReplaceQueryCondition(model interface{}, query string, newCond CondExpr) (string, error) {
	if query == "" {
		return query, newError("Passed empty query for changing condition")
	}

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

	oldCond, err := GetConditionByName(fieldsMap, query, newCond.FieldName, false)
	if err != nil {
		return "", newError("Condition with passed name " + newCond.FieldName + " not found")
	}
	if oldCond == nil { // If condition with passed name not found, exit
		return query, nil
	}

	var oldCondString string
	if oldCond.IsBracket {
		oldCondString = "(" + oldCond.FieldName + oldCond.Operator + fmt.Sprintf("%v", oldCond.Value) + ")"
	} else {
		oldCondString = oldCond.FieldName + oldCond.Operator + fmt.Sprintf("%v", oldCond.Value)
	}

	var newCondString string
	if newCond.IsBracket {
		newCondString = "(" + newCond.FieldName + newCond.Operator + fmt.Sprintf("%v", newCond.Value) + ")"
	} else {
		newCondString = newCond.FieldName + newCond.Operator + fmt.Sprintf("%v", newCond.Value)
	}

	replacer := strings.NewReplacer(oldCondString, newCondString)
	return replacer.Replace(query), nil
}

// DeleteQueryCondition prunes condition from query by fieldname
func DeleteQueryCondition(model interface{}, query, condName string) (string, error) {
	if query == "" {
		return query, newError("Passed empty query for condition prune")
	}

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

	c, _ := GetConditionByName(fieldsMap, query, condName, false)
	if c == nil { // If condition with passed name not found, exit
		return query, nil
	}

	var condString string
	if c.IsBracket {
		condString = "(" + c.FieldName + c.Operator + fmt.Sprintf("%v", c.Value) + ")"
	} else {
		condString = c.FieldName + c.Operator + fmt.Sprintf("%v", c.Value)
	}

	// Prune condition
	for k := range logicalBindings {
		query = strings.Replace(query, k + condString, "", 1)
	}
	query = strings.Replace(query, condString, "", 1)

	// Prune redundant logical operators in conditions set
	for k := range logicalBindings {
		query = strings.Replace(query, "?"+ k, "?", 1)
		query = strings.Replace(query, k + "?", "?", 1)
	}

	return query, nil
}

// AddQueryRestrictions adds restrictions to query restrictions block instead of current
// If argument is not passed, query saves current parameter
func AddQueryRestrictions(query string, sortFields string, sortOrder string, limit string, offset string) (string, error) {
	if query == "" {
		return query, newError("Passed empty query for forming restrictions block")
	}
	queryBlocks := strings.Split(query, "?")

	// If rests block is empty, imitate block structure
	var currentRests []string
	if queryBlocks[2] == "" {
		currentRests = []string{"", "", "", ""}
	} else {
		currentRests = strings.Split(queryBlocks[2], ",")
	}

	// sortFields
	if sortFields != "" {
		currentRests[0] = sortFields
	}
	// sortOrder
	if sortOrder != "" && (sortOrder == "asc" || sortOrder == "desc") {
		currentRests[1] = sortOrder
	}
	// limit
	if limit != "" && !strings.Contains(limit, "-") {
		currentRests[2] = limit
	}
	// offset
	if offset != "" && !strings.Contains(offset, "-") {
		currentRests[3] = offset
	}
	queryBlocks[2] = strings.Join(currentRests, ",")

	return strings.Join(queryBlocks, "?"), nil
}

func extractQueryCondition(fieldsMap map[string]string, cond string, toDBFormat bool) (condExpr *CondExpr, err error) {
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

	if !toDBFormat {
		return &CondExpr{
			FieldName: f,
			Operator:  op,
			Value:     strings.Split(cond, op)[1],
			IsBracket: isBracket,
		}, nil
	}

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
