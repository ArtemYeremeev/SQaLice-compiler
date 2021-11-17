package compiler

import (
	"regexp"
	"strconv"
	"strings"
)

var operatorBindings = map[string]string{
	"==": "=",  // EQUALS
	"!=": "!=", // NOT EQUALS
	"<":  "<",  // LESS
	"<=": "<=", // LESS OR EQUALS
	">":  ">",  // GREATER
	">=": ">=", // GREATER OR EQUALS
	">>": "&&", // OVERLAPS
}

var logicalBindings = map[string]string{
	"*":  "and", // AND
	"||": "or",  // OR
}

// Get builds a GET query with parameters
func Get(model interface{}, target string, params string, withCount bool) (string, string, error) {
	return compile(model, target, params, withCount, "")
}

// Search builds a GET query with LIKE filter on searchField
func Search(model interface{}, target string, params string, withCount bool, searchParams string) (string, string, error) {
	return compile(model, target, params, withCount, searchParams)
}

// compile assembles a query strings to PG database for main query and count query
func compile(model interface{}, target string, params string, withCount bool, searchParams string) (string, string, error) {
	if params == "" {
		return "", "", newError("Request parameters is not passed")
	}

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

	queryBlocks := strings.Split(params, "?")
	selectBlock, err := combineFields(fieldsMap, queryBlocks[0])
	if err != nil {
		return "", "", err
	}

	fromBlock, err := combineTarget(target)
	if err != nil {
		return "", "", err
	}

	whereBlock, err := combineConditions(fieldsMap, queryBlocks[1], searchParams)
	if err != nil {
		return "", "", err
	}

	limitsBlock, err := combineRestrictions(fieldsMap, queryBlocks[2])
	if err != nil {
		return "", "", err
	}

	var respArray []string
	queryArray := []string{selectBlock, fromBlock, whereBlock, limitsBlock}
	for _, block := range queryArray {
		if block == "" {
			continue
		}

		respArray = append(respArray, strings.TrimSpace(block))
	}

	var countQuery string
	mainQuery := strings.Join(respArray, " ")
	if withCount { // compile query to get count of result rows
		q := strings.TrimSpace(strings.Join([]string{"select 1", fromBlock, whereBlock}, " "))
		countQuery = "select count(*) from (" + q + ") q"
	}

	return mainQuery, countQuery, nil
}

// combineSelect assembles SELECT query block
func combineFields(fieldsMap map[string]string, fields string) (string, error) {
	selectBlock := "select "

	var preparedFields []string
	if fields == "" { // Request all model fields
		keys := sortMap(fieldsMap)
		for _, k := range keys {
			preparedField := "q." + fieldsMap[k]
			preparedFields = append(preparedFields, preparedField)
		}
	} else { // Request specific fields from query
		fields := strings.Split(fields, ",")
		for _, f := range fields {
			field := fieldsMap[strings.TrimSpace(f)]
			if field == "" {
				return "", newError("Passed unexpected field name in select - " + f)
			}

			preparedField := "q." + field
			preparedFields = append(preparedFields, preparedField)
		}
	}
	selectBlock = selectBlock + strings.Join(preparedFields, ", ")

	return selectBlock, nil
}

// combineTarget assembles FROM query block
func combineTarget(target string) (string, error) {
	if target == "" {
		return "", newError("Request target not passed")
	}

	return "from " + target + " q", nil
}

// combineConditions assembles WHERE query block
func combineConditions(fieldsMap map[string]string, conds string, searchParams string) (string, error) {
	if conds == "" && searchParams == "" {
		return "", nil
	}

	whereBlock := "where "
	// searchQuery handling
	if searchParams != "" {
		searchConds, err := formSearchConditions(fieldsMap, searchParams)
		if err != nil {
			return "", err
		}
		if searchConds != "" && conds != "" {
			whereBlock = whereBlock + searchConds + "and "
		} else if searchConds != "" {
			whereBlock = whereBlock + searchConds
		}
	}

	// Parse logical operators
	// Get substrings with bracket conditions
	re := regexp.MustCompile(`\((.*?)\)`)
	bracketSubstrings := re.FindAllString(conds, -1)
	var preparedConditions []string
	for _, brCondSet := range bracketSubstrings {
		condSet := brCondSet
		condSet = strings.Trim(condSet, "(")
		condSet = strings.Trim(condSet, ")")

		var (
			bracketConditions []string
			cond              string
			err               error
		)
		opCount := strings.Count(condSet, "*") + strings.Count(condSet, "||")
		for i := 0; i <= opCount; i++ { // loop number of logical operators in condition set
			condSet, cond, err = handleConditionsSet(fieldsMap, condSet)
			if err != nil {
				return "", err
			}
			bracketConditions = append(bracketConditions, cond)
		}
		conds = strings.TrimPrefix(conds, brCondSet)

		// Handle trailing logical operator
		orIndex := strings.Index(conds, "||")
		andIndex := strings.Index(conds, "*")

		op := ""
		if (orIndex < 0 || andIndex < orIndex) && andIndex >= 0 { // handle AND logical condition
			op = "*"
		}
		if (andIndex < 0 || orIndex < andIndex) && orIndex >= 0 { // handle OR logical condition
			op = "||"
		}
		if op != "" {
			conds = strings.TrimPrefix(conds, op)
		}

		preparedConditions = append(preparedConditions, "("+strings.Join(bracketConditions, " ")+") "+logicalBindings[op])
	}

	var cond string
	var err error
	opCount := strings.Count(conds, "*") + strings.Count(conds, "||")
	if conds != "" { // handle non-bracket conditions set
		for i := 0; i <= opCount; i++ { // loop number of logical operators in condition set
			conds, cond, err = handleConditionsSet(fieldsMap, conds)
			if err != nil {
				return "", err
			}
			preparedConditions = append(preparedConditions, cond)
		}
	}

	return whereBlock + strings.Join(preparedConditions, " "), nil
}

// combineRestrictions assembles selection parameters
func combineRestrictions(fieldsMap map[string]string, rests string) (string, error) {
	if rests == "" {
		return "", nil
	}
	restsArray := strings.Split(rests, ",")
	restsBlock := ""

	// field
	field := restsArray[0]
	if field != "" {
		f := fieldsMap[field]
		if f == "" {
			return "", newError("Unexpected selection order field - " + field)
		}
		restsBlock = "order by q." + f + " "
	}

	// order
	order := restsArray[1]
	if order != "" {
		if order != "asc" && order != "desc" {
			return "", newError("Unexpected selection order - " + order)
		}

		if restsBlock == "" {
			restsBlock = "order by q.ID " + order
		} else {
			restsBlock = restsBlock + order
		}
	}

	// limit
	limit := restsArray[2]
	if limit != "" {
		n, err := strconv.Atoi(limit)
		if err != nil {
			return "", newError("Unexpected selection limit - " + limit)
		}
		if n < 0 {
			return "", newError("Invaild negative selection limit - " + limit)
		}

		if restsBlock == "" {
			restsBlock = "limit " + limit
		} else {
			restsBlock = restsBlock + " limit " + limit
		}
	}

	// offset
	offset := restsArray[3]
	if offset != "" {
		n, err := strconv.Atoi(offset)
		if err != nil {
			return "", newError("Unexpected selection offset - " + offset)
		}
		if n < 0 {
			return "", newError("Invaild negative selection offset - " + offset)
		}

		if restsBlock == "" {
			restsBlock = "offset " + offset
		} else {
			restsBlock = restsBlock + " offset " + offset
		}
	}

	return restsBlock, nil
}

func handleConditionsSet(fieldsMap map[string]string, condSet string) (string, string, error) {
	var cond string
	var err error

	orIndex := strings.Index(condSet, "||")
	andIndex := strings.Index(condSet, "*")

	if orIndex < 0 && andIndex < 0 { // no logical condition
		cond, err = formCondition(fieldsMap, condSet, "")
		if err != nil {
			return "", "", err
		}
	} else if orIndex < 0 || (andIndex < orIndex && andIndex >= 0) { // handle AND logical condition
		cond, err = formCondition(fieldsMap, condSet[:strings.Index(condSet, "*")], "*")
		if err != nil {
			return "", "", err
		}
		condSet = strings.TrimPrefix(condSet, condSet[:strings.Index(condSet, "*")]+"*")
	} else { // handle OR logical condition
		cond, err = formCondition(fieldsMap, condSet[:strings.Index(condSet, "||")], "||")
		if err != nil {
			return "", "", err
		}
		condSet = strings.TrimPrefix(condSet, condSet[:strings.Index(condSet, "||")]+"||")
	}

	return condSet, cond, nil
}

// formSearchConditions builds a conditions block with LIKE operator for search
func formSearchConditions(fieldsMap map[string]string, params string) (string, error) {
	condArr := strings.Split(params, "||") // Split by OR operator
	if condArr == nil {
		return "", newError("Passed empty search block")
	}

	resultBlock := "("
	for i, c := range condArr {
		if i != 0 { // Adds logical delimiter between search conditions
			resultBlock = resultBlock + " or "
		}

		condParts := strings.Split(c, "~~")
		if condParts == nil {
			continue
		}

		f := fieldsMap[condParts[0]]
		if f == "" {
			return "", newError("Passed unexpected field name in search condition - " + condParts[0])
		}

		resultBlock = resultBlock + "q." + f + `::text like '%%` + strings.ToLower(condParts[1]) + `%%'`
	}

	return resultBlock + ") ", nil
}

// formCondition builds condition with standart operator
func formCondition(fieldsMap map[string]string, cond string, logicalOperator string) (string, error) {
	var sep string
	for queryOp := range operatorBindings { // Check is condition legal
		if strings.Contains(cond, queryOp) {
			sep = queryOp
		}
		if strings.Contains(cond, queryOp+"=") {
			sep = queryOp + "="
			break
		}
		if strings.Contains(cond, queryOp+">") {
			sep = queryOp + ">"
			break
		}
	}
	if sep == "" {
		return "", newError("Unsupported operator in condition - " + cond)
	}

	f := strings.Split(cond, sep)[0]
	value := strings.Split(cond, sep)[1]

	field := fieldsMap[f]
	if field == "" {
		return "", newError("Passed unexpected field name in condition - " + f)
	}

	var valueType string
	if value == "false" || value == "true" { // handle boolean type
		valueType = "BOOL"
	}
	if valueType == "" {
		_, err := strconv.Atoi(value) // handle integer type
		if err == nil {
			valueType = "INT"
		}
	}
	var arrValue string
	if valueType == "" && strings.Contains(value, ",") { // handle array type
		arrValues := strings.Split(value, ",")
		for _, v := range arrValues {
			_, err := strconv.ParseBool(v)
			if err == nil {
				arrValue = arrValue + v + ","
				continue
			}
			_, err = strconv.Atoi(v)
			if err == nil {
				arrValue = arrValue + v + ","
				continue
			}
			arrValue = arrValue + addPGQuotes(v) + ","
		}
		valueType = "ARRAY"
	}
	if valueType == "" { // string format
		value = addPGQuotes(value)
	}

	switch operatorBindings[sep] { // switch operators
	case "&&": // handle OVERLAPS operator
		switch valueType {
		case "ARRAY": // array format
			cond = field + " " + operatorBindings[sep] + " array[" + strings.TrimRight(arrValue, ",") + "]"
		default: // others
			cond = field + " " + operatorBindings[sep] + " array[" + value + "]"
		}
	default: // rest of operators
		switch valueType {
		case "ARRAY": // array format
			cond = field + " " + operatorBindings[sep] + " any(array[" + strings.TrimRight(arrValue, ",") + "])"
		default: // others
			cond = field + " " + operatorBindings[sep] + " " + value
		}
	}

	if logicalOperator != "" {
		return "q." + cond + " " + logicalBindings[logicalOperator], nil
	}

	return "q." + cond, nil
}
