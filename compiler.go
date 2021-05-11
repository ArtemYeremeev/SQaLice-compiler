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
}

var logicalBindings = map[string]string{
	"*":  "and", // AND
	"||": "or",  // OR
}

// Compile assembles a query strings to PG database for main query and count query
func Compile(modelsMap map[string]map[string]string, target string, params string, withCount bool) (string, string, error) {
	if params == "" {
		return "", "", newError("Request parameters not passed")
	}
	if target == "" {
		return "", "", newError("Request target not passed")
	}

	queryBlocks := strings.Split(params, "?")
	selectBlock, err := combineFields(modelsMap[target], queryBlocks[0])
	if err != nil {
		return "", "", err
	}

	fromBlock, err := combineTarget(target)
	if err != nil {
		return "", "", err
	}

	whereBlock, err := combineConditions(modelsMap[target], queryBlocks[1])
	if err != nil {
		return "", "", err
	}

	limitsBlock, err := combineRestrictions(modelsMap[target], queryBlocks[2])
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
		countQuery = "select count(*) from (" + strings.Join([]string{selectBlock, fromBlock, whereBlock}, " ") + ") q"
	}

	return mainQuery, countQuery, nil
}

// combineSelect assembles SELECT query block
func combineFields(fieldsMap map[string]string, fields string) (string, error) {
	selectBlock := "select "

	if fields == "" {
		selectBlock = selectBlock + "*"
	} else {
		var preparedFields []string

		fields := strings.Split(fields, ",")
		for _, f := range fields {
			field := fieldsMap[strings.TrimSpace(f)]
			if field == "" {
				return "", newError("Passed unexpected field name in select - " + f)
			}

			preparedField := "q." + field
			preparedFields = append(preparedFields, preparedField)
		}

		selectBlock = selectBlock + strings.Join(preparedFields, ", ")
	}

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
func combineConditions(fieldsMap map[string]string, conds string) (string, error) {
	if conds == "" {
		return "", nil
	}

	whereBlock := "where "
	var preparedConditions []string

	// Parse logical operators
	// Get substrings with bracket conditions
	re := regexp.MustCompile(`\((.*?)\)`)
	bracketSubstrings := re.FindAllString(conds, -1)
	for _, brCondSet := range bracketSubstrings {
		condSet := brCondSet

		condSet = strings.Trim(condSet, "(")
		condSet = strings.Trim(condSet, ")")

		var cond string
		var err error

		var bracketConditions []string
		opCount := strings.Count(condSet, "*") + strings.Count(condSet, "||")
		for i := 0; i <= opCount; i++ { // loop number of logical operators in condition set
			condSet, cond, err = handleConditionsSet(fieldsMap, condSet)
			if err != nil {
				return "", err
			}
			bracketConditions = append(bracketConditions, cond)
		}
		conds = strings.TrimPrefix(conds, brCondSet)

		// Handle trailimg logical operator
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
		_, err := strconv.Atoi(limit)
		if err != nil {
			return "", newError("Unexpected selection limit - " + limit)
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
		_, err := strconv.Atoi(offset)
		if err != nil {
			return "", newError("Unexpected selection offset - " + offset)
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
	}

	if sep == "" {
		return "", newError("Unsupported operator in condition " + cond)
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

	switch valueType {
	case "": // default string format
		cond = field + operatorBindings[sep] + addPGQuotes(value)
	case "ARRAY": // array format
		cond = field + " " + operatorBindings[sep] + " any(array[" + strings.TrimRight(arrValue, ",") + "])"
	default: // others
		cond = field + operatorBindings[sep] + value
	}

	if logicalOperator != "" {
		return "q." + cond + " " + logicalBindings[logicalOperator], nil
	}

	return "q." + cond, nil
}
