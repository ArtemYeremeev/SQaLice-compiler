package compiler

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

var testGetFieldsListCases = []struct {
	// Query params
	Query string
	// Response
	FieldsList []string
	Err        error
}{
	{ // 1. Test single query field
		Query:      "ID??",
		FieldsList: []string{"id"},
	},
	{ // 2. Test multiple query fields
		Query:      "ID,content,count??",
		FieldsList: []string{"id", "content", "count"},
	},
	{ // 3. Test multiple query fields with SQL cast
		Query:      "ID,isBool??",
		FieldsList: []string{"id", "is_bool"},
	},
	{ // 4. Test query with empty select block
		Query:      "??",
		FieldsList: nil,
	},
	{ // 5. Test ERROR empty query
		Query:      "",
		FieldsList: nil,
		Err:        newError("Query string not passed"),
	},
	{ // 6. Test ERROR unexpected fieldName in query select block
		Query:      "randomField??",
		FieldsList: nil,
		Err:        newError("Passed unexpected field name in select - randomField"),
	},
}

func TestGetFieldsList(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetFieldsListCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			fieldsList, err := GetFieldsList(m["v_test"], c.Query)
			if err != nil && err.Error() != c.Err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if !isSlicesEqual(fieldsList, c.FieldsList) {
				t.Errorf("expected: %v, got: %v", c.FieldsList, fieldsList)
				t.Fail()
			}
		})
	}
}

func isSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

var testGetConditionsListCases = []struct {
	// Query params
	Query string
	// Response
	CondExprsList []*CondExpr
	Err           error
}{
	{ // 1. Test single condition extraction
		Query:         "?ID==1?",
		CondExprsList: []*CondExpr{{FieldName: "id", Operator: "=", Value: "1", IsBracket: false}},
	},
	{ // 2. Test multiple conditions extraction
		Query:         "?ID>=1*count!=2?",
		CondExprsList: []*CondExpr{{FieldName: "id", Operator: ">=", Value: "1", IsBracket: false}, {FieldName: "count", Operator: "!=", Value: "2", IsBracket: false}},
	},
	{ // 3. Test query with complex conditions block
		Query: "?(ID>>1,2,3)*content==testText*count!=2?",
		CondExprsList: []*CondExpr{
			{FieldName: "id", Operator: "&&", Value: "1,2,3", IsBracket: true},
			{FieldName: "content", Operator: "=", Value: "testText", IsBracket: false},
			{FieldName: "count", Operator: "!=", Value: "2", IsBracket: false},
		},
	},
	{ // 4. Test query with empty conds block
		Query:         "??",
		CondExprsList: nil,
	},
	{ // 5. Test ERROR empty query
		Query:         "",
		CondExprsList: nil,
		Err:           newError("Query string not passed"),
	},
}

func TestGetConditionsList(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetConditionsListCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			condsList, err := GetConditionsList(m["v_test"], c.Query, true)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			for i, cond := range condsList {
				if cond == nil {
					continue
				}
				// Compare struct names
				if cond.FieldName != c.CondExprsList[i].FieldName {
					t.Errorf("Array element "+fmt.Sprint(i)+": expected struct fieldName: %v, got: %v", c.CondExprsList[i].FieldName, cond.FieldName)
					t.FailNow()
				}
				// Compare struct operators
				if cond.Operator != c.CondExprsList[i].Operator {
					t.Errorf("Array element "+fmt.Sprint(i)+": expected struct operator: %v, got: %v", c.CondExprsList[i].Operator, cond.Operator)
					t.FailNow()
				}
				// Compare struct values
				if cond.Value != c.CondExprsList[i].Value {
					t.Errorf("Array element "+fmt.Sprint(i)+": expected struct value: %v, got: %v", c.CondExprsList[i].Value, cond.Value)
					t.FailNow()
				}
				// Compare struct isBracket
				if cond.IsBracket != c.CondExprsList[i].IsBracket {
					t.Errorf("Array element "+fmt.Sprint(i)+": expected struct isBracket: %v, got: %v", c.CondExprsList[i].IsBracket, cond.IsBracket)
					t.FailNow()
				}
			}
		})
	}
}

var testGetConditionByNameCases = []struct {
	// Query params
	Query     string
	FieldName string
	// Response
	CondExpr *CondExpr
	Err      error
}{
	{ // 1. Test single condition extraction
		Query:     "?ID==1?",
		FieldName: "ID",
		CondExpr:  &CondExpr{FieldName: "id", Operator: "=", Value: "1", IsBracket: false},
	},
	{ // 2. Test condition extraction from multiple set
		Query:     "?ID>=1*count!=2?",
		FieldName: "count",
		CondExpr:  &CondExpr{FieldName: "count", Operator: "!=", Value: "2", IsBracket: false},
	},
	{ // 3. Test query with complex conditions block
		Query:     "?(ID>>1,2,3)*content==testText*count!=2?",
		FieldName: "ID",
		CondExpr:  &CondExpr{FieldName: "id", Operator: "&&", Value: "1,2,3", IsBracket: true},
	},
	{ // 4. Test query with empty conds block
		Query:     "??",
		FieldName: "ID",
		CondExpr:  nil,
	},
	{ // 5. Test ERROR query without fieldName
		Query:     "?count!=2?",
		FieldName: "",
		CondExpr:  nil,
		Err:       newError("Condition field name not passed"),
	},
	{ // 6. Test query with empty conds block
		Query:     "?ID^3?",
		FieldName: "ID",
		CondExpr:  nil,
		Err:       newError("Unsupported operator in condition"),
	},
}

func TestGetConditionByName(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetConditionByNameCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			cond, err := GetConditionByName(m["v_test"], c.Query, c.FieldName, true)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			// Handle empty struct
			if cond == nil && c.CondExpr != nil {
				t.Errorf("Expected struct: %v, got: %v", c.CondExpr, cond)
				t.FailNow()
			}
			if cond != nil && c.CondExpr != nil {
				// Compare struct names
				if cond.FieldName != c.CondExpr.FieldName {
					t.Errorf("Expected struct fieldName: %v, got: %v", c.CondExpr.FieldName, cond.FieldName)
					t.FailNow()
				}
				// Compare struct operators
				if cond.Operator != c.CondExpr.Operator {
					t.Errorf("Expected struct operator: %v, got: %v", c.CondExpr.Operator, cond.Operator)
					t.FailNow()
				}
				// Compare struct values
				if cond.Value != c.CondExpr.Value {
					t.Errorf("Expected struct value: %v, got: %v", c.CondExpr.Value, cond.Value)
					t.FailNow()
				}
				// Compare struct isBracket
				if cond.IsBracket != c.CondExpr.IsBracket {
					t.Errorf("Expected struct isBracket: %v, got: %v", c.CondExpr.IsBracket, cond.IsBracket)
					t.FailNow()
				}
			}
		})
	}
}

var testGetRestsCases = []struct {
	// Query params
	Query string
	// Response
	Field  string
	Order  string
	Limit  int
	Offset int
	Err    error
}{
	{ // 1. Test query with full rests block
		Query:  "??ID,asc,10,0",
		Field:  "id",
		Order:  "asc",
		Limit:  10,
		Offset: 0,
	},
}

func TestGetSortField(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetRestsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			field, err := GetSortField(m["v_test"], c.Query)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			// Handle empty struct
			if field != c.Field {
				t.Errorf("Expected sort field: %v, got: %v", c.Field, field)
				t.FailNow()
			}
		})
	}
}

func TestGetSortOrder(t *testing.T) {
	for index, c := range testGetRestsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			order, err := GetSortOrder(c.Query)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			// Handle empty struct
			if order != c.Order {
				t.Errorf("Expected sort field: %v, got: %v", c.Order, order)
				t.FailNow()
			}
		})
	}
}

func TestGetLimit(t *testing.T) {
	for index, c := range testGetRestsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			limit, err := GetLimit(c.Query)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if limit != nil {
				if *limit != c.Limit {
					t.Errorf("Expected limit: %v, got: %v", c.Limit, limit)
					t.FailNow()
				}
			}
			if limit == nil && c.Limit != 0 {
				t.Errorf("Expected limit: %v, got: %v", c.Limit, limit)
				t.FailNow()
			}
		})
	}
}

func TestGetOffset(t *testing.T) {
	for index, c := range testGetRestsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			offset, err := GetOffset(c.Query)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if offset != nil {
				if *offset != c.Offset {
					t.Errorf("Expected sort offset: %v, got: %v", c.Offset, offset)
					t.FailNow()
				}
			}
			if offset == nil && c.Offset != 0 {
				t.Errorf("Expected sort offset: %v, got: %v", c.Offset, offset)
				t.FailNow()
			}
		})
	}
}

var testAddQueryFieldsToSelectCases = []struct {
	// Params
	Query           string
	NewFields       []string
	isDeleteCurrent bool
	// Response
	RespQuery string
	Err       error
}{
	{ // 1. Test adding fields to empty select block
		Query:           "??",
		NewFields:       []string{"ID", "count", "randomField"},
		isDeleteCurrent: true,
		RespQuery:       "ID,count??",
	},
	{ // 2. Test adding fields to select block without deleting current fields
		Query:           "isBool??",
		NewFields:       []string{"ID", "count"},
		isDeleteCurrent: false,
		RespQuery:       "isBool,ID,count??",
	},
	{ // 3. Test adding fields to select block with deleting current fields
		Query:           "isBool??",
		NewFields:       []string{"ID", "count"},
		isDeleteCurrent: true,
		RespQuery:       "ID,count??",
	},
}

func TestAddQueryFieldsToSelect(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testAddQueryFieldsToSelectCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			respQuery, err := AddQueryFieldsToSelect(m["v_test"], c.Query, c.NewFields, c.isDeleteCurrent)
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
				t.FailNow()
			}

			if respQuery != c.RespQuery {
				t.Errorf("Expected response query: %v, got: %v", c.RespQuery, respQuery)
				t.FailNow()
			}
		})
	}
}

var testAddQueryConditionsCases = []struct {
	// Params
	Query           string
	NewConds        []CondExpr
	IsDeleteCurrent bool
	// Response
	RespQuery string
	Err       error
}{
	{ // 1. Test adding 1 condition to empty conditions block
		Query: "??",
		NewConds: []CondExpr{
			{
				FieldName: "ID",
				Operator:  "==",
				Value:     1,
				IsBracket: false,
			},
		},
		IsDeleteCurrent: false,
		RespQuery:       "?ID==1?",
	},
	{ // 2. Test adding 2 conditions to empty conditions block
		Query: "ID??ID,asc,10,0",
		NewConds: []CondExpr{
			{
				FieldName: "ID",
				Operator:  "==",
				Value:     1,
				IsBracket: true,
			},
			{
				FieldName: "count",
				Operator:  "!=",
				Value:     12,
				IsBracket: false,
			},
		},
		IsDeleteCurrent: false,
		RespQuery:       "ID?(ID==1)*count!=12?ID,asc,10,0",
	},
	{ // 3. Test adding condition to conditions block without deleting current
		Query: "ID?isBool==true?ID,asc,10,0",
		NewConds: []CondExpr{
			{
				FieldName: "ID",
				Operator:  "==",
				Value:     1,
				IsBracket: false,
			},
		},
		IsDeleteCurrent: false,
		RespQuery:       "ID?isBool==true*ID==1?ID,asc,10,0",
	},
	{ // 4. Test adding condition to conditions block with deleting current
		Query: "ID?isBool==true?ID,asc,10,0",
		NewConds: []CondExpr{
			{
				FieldName: "isBool",
				Operator:  "==",
				Value:     false,
				IsBracket: true,
			},
		},
		IsDeleteCurrent: true,
		RespQuery:       "ID?(isBool==false)?ID,asc,10,0",
	},
	{ // 5. Test ERROR
		Query: "ID?isBool==true?ID,asc,10,0",
		NewConds: []CondExpr{
			{
				FieldName: "isBool",
				Operator:  "^=",
				Value:     false,
				IsBracket: true,
			},
		},
		IsDeleteCurrent: true,
		RespQuery:       "ID?isBool==true?ID,asc,10,0",
		Err:             newError("Passed incorrect operator in query condition - ^="),
	},
}

func TestAddQueryConditions(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testAddQueryConditionsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			respQuery, err := AddQueryConditions(c.Query, c.NewConds, c.IsDeleteCurrent)
			if err != nil && c.Err.Error() != err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if respQuery != c.RespQuery {
				t.Errorf("Expected response query: %v, got: %v", c.RespQuery, respQuery)
				t.FailNow()
			}
		})
	}
}

var testReplaceQueryConditionCases = []struct {
	// Params
	Query   string
	NewCond CondExpr
	// Response
	RespQuery string
}{
	{ // 1. Test replace condition in conditions block with one condition
		Query: "?ID==1?",
		NewCond: CondExpr{
			FieldName: "ID",
			Operator:  "!=",
			Value:     5,
			IsBracket: true,
		},
		RespQuery: "?(ID!=5)?",
	},
	{ // 2. Test replace condition in conditions block with multiple conditions
		Query: "ID?(ID==1)*count>2?ID,asc,10,0",
		NewCond: CondExpr{
			FieldName: "count",
			Operator:  "!=",
			Value:     12,
			IsBracket: false,
		},
		RespQuery: "ID?(ID==1)*count!=12?ID,asc,10,0",
	},
	{ // 3. Test replace condition in conditions block without suitable condition
		Query: "ID?isBool==true?ID,asc,10,0",
		NewCond: CondExpr{
			FieldName: "ID",
			Operator:  "==",
			Value:     1,
			IsBracket: false,
		},
		RespQuery: "ID?isBool==true?ID,asc,10,0",
	},
}

func TestReplaceQueryCondition(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testReplaceQueryConditionCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			respQuery, err := ReplaceQueryCondition(m["v_test"], c.Query, c.NewCond)
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
				t.FailNow()
			}

			if respQuery != c.RespQuery {
				t.Errorf("Expected response query: %v, got: %v", c.RespQuery, respQuery)
				t.FailNow()
			}
		})
	}
}

var testAddQueryRestrictionsCases = []struct {
	// Params
	Query     string
	SortField string
	SortOrder string
	Limit     string
	Offset    string
	// Response
	RespQuery string
}{
	{ // 1. Test add sortField to rests block
		Query:     "ID,count??",
		SortField: "ID",
		RespQuery: "ID,count??ID,,,",
	},
	{ // 2. Test add sortField and sortOrder to rests block
		Query:     "ID,count??",
		SortField: "ID",
		SortOrder: "desc",
		RespQuery: "ID,count??ID,desc,,",
	},
	{ // 3. Test add limit and offset to rests block
		Query:     "ID,count??",
		Limit:     "10",
		Offset:    "2",
		RespQuery: "ID,count??,,10,2",
	},
	{ // 4. Test add all rests to rests block
		Query:     "ID,count??",
		SortField: "count",
		SortOrder: "asc",
		Limit:     "10",
		Offset:    "2",
		RespQuery: "ID,count??count,asc,10,2",
	},
	{ // 5. Test replace all rests to rests block
		Query:     "ID,count??ID,,5,0",
		SortField: "count",
		SortOrder: "asc",
		Limit:     "10",
		Offset:    "2",
		RespQuery: "ID,count??count,asc,10,2",
	},
}

func TestAddQueryRestrictions(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = formDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testAddQueryRestrictionsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			respQuery, err := AddQueryRestrictions(c.Query, c.SortField, c.SortOrder, c.Limit, c.Offset)
			if err != nil {
				t.Errorf("expected err: %v , got: %v", nil, err)
				t.FailNow()
			}

			if respQuery != c.RespQuery {
				t.Errorf("Expected response query: %v , got: %v", c.RespQuery, respQuery)
				t.FailNow()
			}
		})
	}
}
