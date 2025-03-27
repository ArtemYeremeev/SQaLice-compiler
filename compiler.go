package compiler

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

// Logical bindings between SQaLice and PG
var logicalBindings = map[string]string{
	"*":  "and", // AND
	"||": "or",  // OR
}

// Standart bindings between SQaLice and PG
var operatorBindings = map[string]string{
	"==":  "=",   // EQUALS
	"!=":  "!=",  // NOT EQUALS
	"<":   "<",   // LESS
	"<=":  "<=",  // LESS OR EQUALS
	">":   ">",   // GREATER
	">=":  ">=",  // GREATER OR EQUALS
	">>":  "&&",  // OVERLAPS
	"->>": "->>", // INCLUDES
}

// Bindings for null field values
var nullOperatorBindings = map[string]string{
	"==": "=",  // EQUALS
	"!=": "!=", // NOT EQUALS
}

// Get builds a GET query with parameters
func Get(model interface{}, target, params string, withCount, withArgs bool) (mainQ, countQ string, args []interface{}, er error) {
	return compile(model, target, params, withCount, withArgs, "")
}

// Search builds a GET query with LIKE filter on searchField
func Search(model interface{}, target, params string, withCount, withArgs bool, searchParams string) (mainQ, countQ string, args []interface{}, er error) {
	return compile(model, target, params, withCount, withArgs, searchParams)
}

// compile assembles a query strings to PG database for main query and count query
func compile(model interface{}, target, params string, withCount, withArgs bool, searchParams string) (mainQ, countQ string, args []interface{}, er error) {
	if params == "" {
		return "", "", nil, newError("Request parameters is not passed")
	}

	// form fields map with formDinamicModel
	fieldsMap := formDinamicModel(model)

	queryBlocks := strings.Split(params, "?")
	selectBlock, err := combineFields(fieldsMap, queryBlocks[0])
	if err != nil {
		return "", "", nil, err
	}

	fromBlock, err := combineTarget(target)
	if err != nil {
		return "", "", nil, err
	}

	whereBlock, args, err := combineConditions(fieldsMap, queryBlocks[1], searchParams, withArgs)
	if err != nil {
		return "", "", nil, err
	}

	limitsBlock, err := combineRestrictions(fieldsMap, queryBlocks[2])
	if err != nil {
		return "", "", nil, err
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

	return mainQuery, countQuery, args, nil
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
func combineConditions(fieldsMap map[string]string, conds, searchParams string, withArgs bool) (string, []interface{}, error) {
	// Prune spaces
	conds = strings.ReplaceAll(conds, " ", "")
	searchParams = strings.ReplaceAll(searchParams, " ", "%")

	if conds == "" && searchParams == "" {
		return "", nil, nil
	}

	var condIndex *int
	if withArgs {
		condIndex = func(i int)*int{return &i}(1)
	}

	whereBlock := "where "
	var searchArgs []interface{}
	if searchParams != "" { // searchQuery handling
		var (
			searchConds string
			err error
		)
		searchConds, searchArgs, condIndex, err = formSearchConditions(fieldsMap, searchParams, condIndex)
		if err != nil {
			return "", nil, err
		}
		if searchConds != "" && conds != "" {
			whereBlock = whereBlock + searchConds + "and "
		} else if searchConds != "" {
			whereBlock = whereBlock + searchConds
		}
	}

	// standart conditions block handling
	preparedConditions, preparedArgs, _, err := extractConditionsSet(fieldsMap, conds, false, condIndex)
	if err != nil {
		return "", nil, err
	}

	return whereBlock + strings.Join(preparedConditions, " "), append(searchArgs, preparedArgs...), nil
}

// combineRestrictions assembles selection parameters
func combineRestrictions(fieldsMap map[string]string, rests string) (string, error) {
	if rests == "" {
		return "", nil
	}
	restsArr := strings.Split(rests, ",")
	restsBlock := ""

	// order
	order := restsArr[1]
	if order != "" {
		if order != "asc" && order != "desc" {
			return "", newError("Unexpected selection order - " + order)
		}
	} else {
		order = "asc"
	}

	// fields
	if restsArr[0] != "" {
		for i, field := range strings.Split(restsArr[0], "|") {
			f := fieldsMap[field]
			if f == "" {
				return "", newError("Unexpected selection order field - " + restsArr[0])
			}

			if i == 0 {
				restsBlock = "order by q." + f + " " + order
			} else {
				restsBlock = restsBlock + ", q." + f + " " + order
			}
		}
	}

	// limit
	limit := restsArr[2]
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
	offset := restsArr[3]
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

// formSearchConditions builds a conditions block with LIKE operator for search
func formSearchConditions(fieldsMap map[string]string, params string, condIndex *int) (string, []interface{}, *int, error) {
	preparedConds, preparedArgs, condI, err := extractConditionsSet(fieldsMap, params, true, condIndex)
	if err != nil {
		return "", nil, nil, err
	}

	return "(" + strings.Join(preparedConds, " ") + ") ", preparedArgs, condI, nil
}

func extractConditionsSet(fieldsMap map[string]string, conds string, isSearch bool, condIndex *int) ([]string, []interface{}, *int, error) {
	if isSearch {
		conds = strings.ReplaceAll(conds, "(", "")
		conds = strings.ReplaceAll(conds, ")", "")
	}
	bracketSubstrings := regexp.MustCompile(`\(.*?(=|~|\|).*?\)`).FindAllString(conds, -1) // \((.*?)\)

	// Parse logical operators
	var (
		arg interface{}
		preparedArgs []interface{}
		preparedConds []string
	)
	for _, brCondSet := range bracketSubstrings {
		condSet := strings.Trim(brCondSet, "(")
		condSet = strings.Trim(condSet, ")")

		var (
			bracketConditions []string
			cond              string
			err               error
		)
		opCount := strings.Count(condSet, "*") + strings.Count(condSet, "||")
		for i := 0; i <= opCount; i++ { // loop number of logical operators in condition set
			condSet, cond, arg, condIndex, err = handleConditionsSet(fieldsMap, condSet, isSearch, condIndex)
			if err != nil {
				return nil, nil, nil, err
			}
			bracketConditions = append(bracketConditions, cond)
			preparedArgs = append(preparedArgs, arg)
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

		brCondSet := "(" + strings.Join(bracketConditions, " ") + ")"
		if op != "" {
			brCondSet = brCondSet + " " + logicalBindings[op]
		}
		preparedConds = append(preparedConds, brCondSet)
	}

	var cond string
	var err error
	opCount := strings.Count(conds, "*") + strings.Count(conds, "||")
	if conds != "" { // handle non-bracket conditions set
		for i := 0; i <= opCount; i++ { // loop number of logical operators in condition set
			conds, cond, arg, condIndex, err = handleConditionsSet(fieldsMap, conds, isSearch, condIndex)
			if err != nil {
				return nil, nil, nil, err
			}
			preparedConds = append(preparedConds, cond)
			preparedArgs = append(preparedArgs, arg)
		}
	}

	return preparedConds, preparedArgs, condIndex, nil
}

func handleConditionsSet(fieldsMap map[string]string, condSet string, isSearch bool, condIndex *int) (string, string, interface{}, *int, error) {
	orIndex := strings.Index(condSet, "||")
	andIndex := strings.Index(condSet, "*")

	var (
		err error
		cond string
		arg interface{}
	)
	if orIndex < 0 && andIndex < 0 { // no logical condition
		cond, arg, condIndex, err = formCondition(fieldsMap, condSet, "", isSearch, condIndex)
		if err != nil {
			return "", "", nil, nil, err
		}
	} else if orIndex < 0 || (andIndex < orIndex && andIndex >= 0) { // handle AND logical condition
		cond, arg, condIndex, err = formCondition(fieldsMap, condSet[:strings.Index(condSet, "*")], "*", isSearch, condIndex)
		if err != nil {
			return "", "", nil, nil, err
		}
		condSet = strings.TrimPrefix(condSet, condSet[:strings.Index(condSet, "*")]+"*")
	} else { // handle OR logical condition
		cond, arg, condIndex, err = formCondition(fieldsMap, condSet[:strings.Index(condSet, "||")], "||", isSearch, condIndex)
		if err != nil {
			return "", "", nil, nil, err
		}
		condSet = strings.TrimPrefix(condSet, condSet[:strings.Index(condSet, "||")]+"||")
	}

	return condSet, cond, arg, condIndex, nil
}

// formCondition builds condition with standart operator
func formCondition(fieldsMap map[string]string, cond, logicalOperator string, isSearch bool, condIndex *int) (cnd string, ar interface{}, ind *int, er error) {
	var arg interface{}
	if isSearch { // handle search condition
		condParts := strings.Split(cond, "~~")
		if condParts == nil {
			return "", nil, nil, nil
		}
		f := fieldsMap[condParts[0]]
		if f == "" {
			return "", nil, nil, newError("Passed unexpected field name in search condition - " + condParts[0])
		}

		// handle nested JSONB search field
		nestedArr := strings.Split(condParts[1], "^^")
		if nestedArr[0] != condParts[1] {
			f = "lower(q." + f + operatorBindings["->>"] + `'` + nestedArr[0] + `'::text) like `
			condParts[1] = nestedArr[1]
		} else {
			f = "lower(q." + f + `::text) like `
		}

		value := "%" + pruneInjections(condParts[1], true) + "%"
		if condIndex != nil {
			v, err := strconv.Atoi(value)
			if err == nil {
				arg = v
			}

			b, err := strconv.ParseBool(value)
			if err == nil {
				arg = b
			}

			if arg == nil {
				arg = strings.ToLower(value)
			}

			value = "$" + strconv.Itoa(*condIndex)
			condIndex = func(i int)*int{i = *condIndex + 1; return &i}(*condIndex)
		}

		if logicalOperator != "" {
			return f + strings.ToLower(value) + " " + logicalBindings[logicalOperator], arg, condIndex, nil
		}

		return f + strings.ToLower(value), arg, condIndex, nil
	}

	var sep string
	for queryOp := range operatorBindings { // check is condition legal
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
		return "", nil, nil, newError("Unsupported operator in condition - " + cond)
	}

	f := strings.Split(cond, sep)[0]
	value := pruneInjections(strings.Split(cond, sep)[1], false)

	field := fieldsMap[f]
	if field == "" {
		return "", nil, nil, newError("Passed unexpected field name in condition - " + f)
	}

	// handle nested JSONB field
	var valueType string
	nestedArr := strings.Split(value, "^^")
	if nestedArr[0] != value {
		field = "q." + f + operatorBindings["->>"] + `'` + nestedArr[0] + `'`
		if strings.Contains(nestedArr[1], ",") || sep == ">>" { // handle nested JSONB array value
			value = handleArrCondValues(nestedArr[1], false)
			valueType = "ARRAY"
		}

		_, err := strconv.Atoi(nestedArr[1])
		if err == nil {
			value = nestedArr[1]
			valueType = "INT"
		}

		_, err = strconv.ParseBool(nestedArr[1])
		if err == nil {
			value = nestedArr[1]
			valueType = "BOOL"
		}

		if valueType == "" {
			value = nestedArr[1]
			valueType = "STRING"
		}
	} else {
		field = "q." + field
	}

	// Handle value type
	switch value {
	case "null", "NULL", "undefined": // NULL
		valueType = "NULL"
	case "false", "true", "FALSE", "TRUE", "True", "False": // BOOLEAN
		valueType = "BOOL"
	default:
		_, err := strconv.Atoi(value) // INTEGER
		if err == nil && sep != ">>" {
			valueType = "INT"
		}

		if valueType == "" && (strings.Contains(value, ",") || sep == ">>") { // ARRAY
			value = handleArrCondValues(value, false)
			valueType = "ARRAY"
		}
		if valueType == "" { // STRING by default
			if len(value) > 48 {
				return "", nil, nil, newError("Too long string value in condition - " + value)
			}
		}
	}
	value = strings.TrimRight(value, ",")

	// handle separate query+args implementation
	if condIndex != nil && valueType != "NULL" {
		arg, value, condIndex = handleArgValue(value, valueType, condIndex)
	}

	switch operatorBindings[sep] { // switch operators
	case "&&": // handle OVERLAPS operator
		switch valueType {
		case "ARRAY": // array format
			cond = field + " " + operatorBindings[sep] + " " + value
		case "NULL": // unexpected null value
			return "", nil, nil, newError("Passed unexpected OVERLAPS operator in NULL condition")
		default: // others
			cond = field + " " + operatorBindings[sep] + " " + value
		}
	default: // rest of operators
		switch valueType {
		case "ARRAY": // array format
			switch operatorBindings[sep] { // handle operators inside array condition
			case "=":
				cond = field + " =" + " any(" + value + ")"
			case "!=":
				cond = "not " + field + " =" + " any(" + value + ")"
			default:
				return "", nil, nil, newError("Passed unexpected operator in array condition - " + sep)
			}
		case "NULL": // null values
			switch nullOperatorBindings[sep] {
			case "=":
				cond = field + " is null"
			case "!=":
				cond = field + " is not null"
			default:
				return "", nil, nil, newError("Passed unexpected operator in NULL condition - " + sep)
			}
		default: // others
			cond = field + " " + operatorBindings[sep] + " " + value
		}
	}

	if logicalOperator != "" {
		return cond + " " + logicalBindings[logicalOperator], arg, condIndex, nil
	}

	return cond, arg, condIndex, nil
}

// handleArgValue ...
func handleArgValue(value, valueType string, condIndex *int) (ar interface{}, val string, ind *int) {
	var arg interface{}
	switch valueType {
	case "ARRAY":
		// Detect array type
		arrValues := strings.Split(value, ",")
		_, err := strconv.Atoi(arrValues[0])
		if err == nil {
			var intArr []int
			for _, el := range arrValues {
				v, _ := strconv.Atoi(el)
				intArr = append(intArr, v)
			}
			arg = pq.Array(intArr)
		} else {
			arg = pq.Array(arrValues)
		}
	case "INT":
		v, _ := strconv.Atoi(value)
		arg = v
	case "BOOL":
		v, _ := strconv.ParseBool(value)
		arg = v
	default:
		arg = value
	}

	value = "$" + strconv.Itoa(*condIndex)
	return arg, value, func(i int)*int{i = *condIndex + 1; return &i}(*condIndex)
}

// handleArrCondValues preprocess values inside query condition
func handleArrCondValues(value string, isNestedJson bool) string {
	arrValues := strings.Split(value, ",")

	var respValue string
	for _, v := range arrValues {
		if isNestedJson {
			respValue = respValue + v + ","
		} else {
			_, err := strconv.ParseBool(v)
			if err == nil {
				respValue = respValue + v + ","
				continue
			}
			_, err = strconv.Atoi(v)
			if err == nil {
				respValue = respValue + v + ","
				continue
			}
			respValue = respValue + v + ","
		}
	}

	return respValue
}

// pruneInjections cleans query params from SQL marks
func pruneInjections(str string, isSearch bool) string {
	if isSearch {
		return regexp.MustCompile(`[^а-яА-Яa-zA-Z0-9',№^ ]+`).ReplaceAllString(str, "%")
	}
	return regexp.MustCompile(`[^а-яА-Яa-zA-Z0-9{}_',-^ ]+`).ReplaceAllString(str, "")
}
