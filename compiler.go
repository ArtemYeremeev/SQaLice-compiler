package compiler

import (
	"regexp"
	"strconv"
	"strings"

	"crypto/aes"
	"crypto/cipher"
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
func Get(model interface{}, target, params string, withCount bool) (mainQ, countQ  string, er error) {
	return compile(model, target, params, withCount, "")
}

// EncryptedGet builds a GET query with encrypted params
func EncryptedGet(model interface{}, target, params string, withCount bool, key string) (mainQ, countQ string, er error) {
	return compile(model, target, decryptParams(params, key), withCount, "")
}

// Search builds a GET query with LIKE filter on searchField
func Search(model interface{}, target, params string, withCount bool, searchParams string) (mainQ, countQ  string, er error) {
	return compile(model, target, params, withCount, searchParams)
}

// EncryptedSearch builds a GET query with LIKE filter on searchField with decrypted params and searchParams
func EncryptedSearch(model interface{}, target, params string, withCount bool, searchParams, key string) (mainQ, countQ  string, er error) {
	return compile(model, target, decryptParams(params, key), withCount, decryptParams(searchParams, key))
}

// decryptParams filter query through AES key
func decryptParams(params, key string) string {
	cp, err := aes.NewCipher([]byte(key))
	if err != nil {
		return params
	}

	c, err := cipher.NewGCM(cp)
	if err != nil {
		return params
	}

	r, err := c.Open(nil, []byte(params)[:c.NonceSize()], []byte(params)[c.NonceSize():], nil)
	if err != nil {
		return params
	}

	return string(r[:])
}

// compile assembles a query strings to PG database for main query and count query
func compile(model interface{}, target string, params string, withCount bool, searchParams string) (mainQ, countQ  string, er error) {
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
	// Prune spaces
	conds = strings.ReplaceAll(conds, " ", "")
	searchParams = strings.ReplaceAll(searchParams, " ", "%")

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

	// standart conditions block handling
	preparedConditions, err := extractConditionsSet(fieldsMap, conds, false)
	if err != nil {
		return "", err
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

// formSearchConditions builds a conditions block with LIKE operator for search
func formSearchConditions(fieldsMap map[string]string, params string) (string, error) {
	preparedConditions, err := extractConditionsSet(fieldsMap, params, true)
	if err != nil {
		return "", err
	}

	return "(" + strings.Join(preparedConditions, " ") + ") ", nil
}

func extractConditionsSet(fieldsMap map[string]string, conds string, isSearch bool) ([]string, error) {
	// Parse logical operators
	bracketSubstrings := regexp.MustCompile(`\((.*?)\)`).FindAllString(conds, -1)
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
			condSet, cond, err = handleConditionsSet(fieldsMap, condSet, isSearch)
			if err != nil {
				return nil, err
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

		brCondSet := "(" + strings.Join(bracketConditions, " ") + ")"
		if op != "" {
			brCondSet = brCondSet + " " + logicalBindings[op]
		}
		preparedConditions = append(preparedConditions, brCondSet)
	}

	var cond string
	var err error
	opCount := strings.Count(conds, "*") + strings.Count(conds, "||")
	if conds != "" { // handle non-bracket conditions set
		for i := 0; i <= opCount; i++ { // loop number of logical operators in condition set
			conds, cond, err = handleConditionsSet(fieldsMap, conds, isSearch)
			if err != nil {
				return nil, err
			}
			preparedConditions = append(preparedConditions, cond)
		}
	}

	return preparedConditions, nil
}

func handleConditionsSet(fieldsMap map[string]string, condSet string, isSearch bool) (string, string, error) {
	var cond string
	var err error

	orIndex := strings.Index(condSet, "||")
	andIndex := strings.Index(condSet, "*")

	if orIndex < 0 && andIndex < 0 { // no logical condition
		cond, err = formCondition(fieldsMap, condSet, "", isSearch)
		if err != nil {
			return "", "", err
		}
	} else if orIndex < 0 || (andIndex < orIndex && andIndex >= 0) { // handle AND logical condition
		cond, err = formCondition(fieldsMap, condSet[:strings.Index(condSet, "*")], "*", isSearch)
		if err != nil {
			return "", "", err
		}
		condSet = strings.TrimPrefix(condSet, condSet[:strings.Index(condSet, "*")]+"*")
	} else { // handle OR logical condition
		cond, err = formCondition(fieldsMap, condSet[:strings.Index(condSet, "||")], "||", isSearch)
		if err != nil {
			return "", "", err
		}
		condSet = strings.TrimPrefix(condSet, condSet[:strings.Index(condSet, "||")]+"||")
	}

	return condSet, cond, nil
}

// formCondition builds condition with standart operator
func formCondition(fieldsMap map[string]string, cond string, logicalOperator string, isSearch bool) (string, error) {
	if isSearch { // handle search condition
		condParts := strings.Split(cond, "~~")
		if condParts == nil {
			return "", nil
		}
		f := fieldsMap[condParts[0]]
		if f == "" {
			return "", newError("Passed unexpected field name in search condition - " + condParts[0])
		}

		// handle nested JSONB search field
		nestedArr := strings.Split(condParts[1], "^^")
		if nestedArr[0] != condParts[1] {
			f = "lower(q." + f + operatorBindings["->>"] + `'` + nestedArr[0] + `'::text) like '%`
			condParts[1] = nestedArr[1]
		} else {
			f = "lower(q." + f + `::text) like '%`
		}

		value := pruneInjections(condParts[1], true)
		if logicalOperator != "" {
			return f + strings.ToLower(value) + `%'` + " " + logicalBindings[logicalOperator], nil
		}

		return f + strings.ToLower(value) + `%'`, nil
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
		return "", newError("Unsupported operator in condition - " + cond)
	}

	f := strings.Split(cond, sep)[0]
	value := pruneInjections(strings.Split(cond, sep)[1], false)

	field := fieldsMap[f]
	if field == "" {
		return "", newError("Passed unexpected field name in condition - " + f)
	}

	// handle nested JSONB field
	var valueType, arrValue string
	nestedArr := strings.Split(value, "^^")
	if nestedArr[0] != value {
		field = "q." + f + operatorBindings["->>"] + `'` + nestedArr[0] + `'`
		if strings.Contains(nestedArr[1], ",") { // handle nested JSONB array value
			arrValue = handleArrCondValues(nestedArr[1], true)
			valueType = "ARRAY"
		} else {
			value = `'` + nestedArr[1] + `'`
			valueType = "STRING"
		}
	} else {
		field = "q." + field
	}

	// Handle value type
	switch value {
	case "null", "NULL": // NULL
		valueType = "NULL"
	case "false", "true", "FALSE", "TRUE": // BOOLEAN
		valueType = "BOOL"
	default:
		_, err := strconv.Atoi(value) // INTEGER
		if err == nil {
			valueType = "INT"
		}

		if valueType == "" && strings.Contains(value, ",") { // ARRAY
			arrValue = handleArrCondValues(value, false)
			valueType = "ARRAY"
		}
		if valueType == "" { // STRING by default
			if len(value) > 32 {
				return "", newError("Too long string value in condition - " + value)
			}

			value = addPGQuotes(value)
		}
	}

	switch operatorBindings[sep] { // switch operators
	case "&&": // handle OVERLAPS operator
		switch valueType {
		case "ARRAY": // array format
			cond = field + " " + operatorBindings[sep] + " array[" + strings.TrimRight(arrValue, ",") + "]"
		case "NULL": // unexpected null value
			return "", newError("Passed unexpected OVERLAPS operator in NULL condition")
		default: // others
			cond = field + " " + operatorBindings[sep] + " array[" + value + "]"
		}
	default: // rest of operators
		switch valueType {
		case "ARRAY": // array format
			switch operatorBindings[sep] { // handle operators inside array condition
			case "=":
				cond = field + " =" + " any(array[" + strings.TrimRight(arrValue, ",") + "])"
			case "!=":
				cond = "not " + field + " =" + " any(array[" + strings.TrimRight(arrValue, ",") + "])"
			default:
				return "", newError("Passed unexpected operator in array condition - " + sep)
			}
		case "NULL": // null values
			switch nullOperatorBindings[sep] {
			case "=":
				cond = field + " is null"
			case "!=":
				cond = field + " is not null"
			default:
				return "", newError("Passed unexpected operator in NULL condition - " + sep)
			}
		default: // others
			cond = field + " " + operatorBindings[sep] + " " + value
		}
	}

	if logicalOperator != "" {
		return cond + " " + logicalBindings[logicalOperator], nil
	}

	return cond, nil
}

// handleArrCondValues preprocess values inside query condition
func handleArrCondValues(value string, isNestedJson bool) string {
	arrValues := strings.Split(value, ",")

	var respValue string
	for _, v := range arrValues {
		if isNestedJson {
			respValue = respValue + addPGQuotes(v) + ","
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
			respValue = respValue + addPGQuotes(v) + ","
		}
	}

	return respValue
}

// pruneInjections cleans query params from SQL marks
func pruneInjections(str string, isSearch bool) string {
	if isSearch {
		return regexp.MustCompile(`[^а-яА-Яa-zA-Z0-9',^ ]+`).ReplaceAllString(str, "%")
	}
	return regexp.MustCompile(`[^а-яА-Яa-zA-Z0-9{}_',-^ ]+`).ReplaceAllString(str, "")
}
