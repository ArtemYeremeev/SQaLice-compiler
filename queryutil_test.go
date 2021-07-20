package main

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
}

func TestGetFieldsList(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = FormDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetFieldsListCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			fieldsList, err := GetFieldsList(m["v_test"], c.Query)
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
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
}

func TestGetConditionsList(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = FormDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetConditionsListCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			condsList, err := GetConditionsList(m["v_test"], c.Query, true)
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
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
}

func TestGetConditionByName(t *testing.T) {
	m := make(map[string]map[string]string, 1)
	m["v_test"] = FormDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetConditionByNameCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			cond, err := GetConditionByName(m["v_test"], c.Query, c.FieldName, true)
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
				t.FailNow()
			}

			// Handle empty struct
			if cond == nil && c.CondExpr != nil {
				t.Errorf("Expected struct: %v, got: %v", c.CondExpr, cond)
				t.FailNow()
			}
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
	m["v_test"] = FormDinamicModel(reflect.ValueOf(TestModel{}))

	for index, c := range testGetRestsCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			field, err := GetSortField(m["v_test"], c.Query)
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
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
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
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
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
				t.FailNow()
			}

			if limit != nil {
				if *limit != c.Limit {
					t.Errorf("Expected limit: %v, got: %v", c.Limit, limit)
					t.FailNow()
				}
			}
			if limit == nil && &c.Limit != nil {
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
			if err != nil {
				t.Errorf("expected err: %v, got: %v", nil, err)
				t.FailNow()
			}

			if offset != nil {
				if *offset != c.Offset {
					t.Errorf("Expected sort offset: %v, got: %v", c.Offset, offset)
					t.FailNow()
				}
			}
			if offset == nil && &c.Offset != nil {
				t.Errorf("Expected sort offset: %v, got: %v", c.Offset, offset)
				t.FailNow()
			}
		})
	}
}
