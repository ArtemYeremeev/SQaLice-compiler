package compiler

import (
	"strconv"
	"testing"
)

type TestModel struct {
	ID      *int64  `json:"ID,omitempty" sql:"id"`
	Content *string `json:"content,omitempty" sql:"content"`
	Count   *int    `json:"count,omitempty" sql:"count"`
	IsBool  *bool   `json:"isBool,omitempty" sql:"is_bool"`
	TestNestedModel
}

type TestNestedModel struct {
	ExtraField   *string `json:"extraField,omitempty" sql:"extra_field"`
	OneMoreField *bool   `json:"oneMoreField,omitempty" sql:"one_more_field"`
}

var testGetCases = []struct {
	// Get params
	ModelsMap map[string]map[string]string
	Target    string
	Params    string
	WithCount bool

	// Get response
	MainQuery  string
	CountQuery string
	Err        error
}{
	{ // 1. Test empty params blocks
		Target:    "v_test",
		Params:    "??",
		WithCount: true,

		MainQuery:  "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
	},
	{ // 2. Test fields params block with 1 field
		Target:    "v_test",
		Params:    "ID??",
		WithCount: true,

		MainQuery:  "select q.id from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
	},
	{ // 3. Test fields params block with 3 fields
		Target:    "v_test",
		Params:    "ID,content,count??",
		WithCount: true,

		MainQuery:  "select q.id, q.content, q.count from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
	},
	{ // 4. Test conditions params block with 1 condition
		Target:    "v_test",
		Params:    "ID?ID==1?",
		WithCount: false,

		MainQuery:  "select q.id from v_test q where q.id = 1",
		CountQuery: "",
	},
	{ // 5. Test conditions params block with 1 non-bracket conditionsSet
		Target:    "v_test",
		Params:    "content?ID==1*ID==3?",
		WithCount: false,

		MainQuery:  "select q.content from v_test q where q.id = 1 and q.id = 3",
		CountQuery: "",
	},
	{ // 6. Test conditions params block with 1 bracket conditionsSet
		Target:    "v_test",
		Params:    "ID?(ID==1||ID==3)?",
		WithCount: false,

		MainQuery:  "select q.id from v_test q where (q.id = 1 or q.id = 3)",
		CountQuery: "",
	},
	{ // 7. Test conditions params block with 2 bracket conditionsSets
		Target:    "v_test",
		Params:    "ID?(ID>1||ID<=3)*(content!=test1*content==test2)?",
		WithCount: false,

		MainQuery:  "select q.id from v_test q where (q.id > 1 or q.id <= 3) and (q.content != 'test1' and q.content = 'test2')",
		CountQuery: "",
	},
	{ // 8. Test conditions params block with 1 bracket conditionsSet and 1 non-bracket conditionsSet
		Target:    "v_test",
		Params:    "ID,count?(count!=1*count!=3)*ID==2?",
		WithCount: false,

		MainQuery:  "select q.id, q.count from v_test q where (q.count != 1 and q.count != 3) and q.id = 2",
		CountQuery: "",
	},
	{ // 9. Test conditions params block with 2 bracket conditionsSet and 1 non-bracket conditionsSet
		Target:    "v_test",
		Params:    "content,count?(count!=1*count!=3||ID>=11)||(content==something awful||content==critical404)*ID!=42?",
		WithCount: false,

		MainQuery:  "select q.content, q.count from v_test q where (q.count != 1 and q.count != 3 or q.id >= 11) or (q.content = 'something awful' or q.content = 'critical404') and q.id != 42",
		CountQuery: "",
	},
	{ // 10. Test conditions params block with 1 non-bracket array condition
		Target:    "v_test",
		Params:    "ID,isBool?ID==1,2,test1?",
		WithCount: false,

		MainQuery:  "select q.id, q.is_bool from v_test q where q.id = any(array[1,2,'test1'])",
		CountQuery: "",
	},
	{ // 11. Test conditions params block with 1 non-bracket array conditionsSet
		Target:    "v_test",
		Params:    "count?ID==1,2,test1*content==test2,true?",
		WithCount: false,

		MainQuery:  "select q.count from v_test q where q.id = any(array[1,2,'test1']) and q.content = any(array['test2',true])",
		CountQuery: "",
	},
	{ // 12 Test conditions params block with OVERLAPS operator and single value
		Target:    "v_test",
		Params:    "ID?content>>value?",
		WithCount: true,

		MainQuery:  "select q.id from v_test q where q.content && array['value']",
		CountQuery: "select count(*) from (select 1 from v_test q where q.content && array['value']) q",
	},
	{ // 13 Test conditions params block with OVERLAPS operator and muliple values
		Target:    "v_test",
		Params:    "ID,count?content>>value1,value2,true,14*ID==25?,,10,0",
		WithCount: true,

		MainQuery:  "select q.id, q.count from v_test q where q.content && array['value1','value2',true,14] and q.id = 25 limit 10 offset 0",
		CountQuery: "select count(*) from (select 1 from v_test q where q.content && array['value1','value2',true,14] and q.id = 25) q",
	},
	{ // 14. Test restrictions params block with all restrictions
		Target:    "v_test",
		Params:    "ID?isBool==true?ID,desc,10,0",
		WithCount: false,

		MainQuery:  "select q.id from v_test q where q.is_bool = true order by q.id desc limit 10 offset 0",
		CountQuery: "",
	},
	{ // 15. Test restrictions params block with order field
		Target:    "v_test",
		Params:    "ID??ID,,,",
		WithCount: false,

		MainQuery:  "select q.id from v_test q order by q.id",
		CountQuery: "select count(*) from (select 1 from v_test q order by q.id) q",
	},
	{ // 16. Test restrictions params block with limit
		Target:    "v_test",
		Params:    "ID,isBool,content?ID!=42?,,5,",
		WithCount: false,

		MainQuery:  "select q.id, q.is_bool, q.content from v_test q where q.id != 42 limit 5",
		CountQuery: "",
	},
	{ // 17. Test restrictions params block with limit and offset
		Target:    "v_test",
		Params:    "ID??,,10,2",
		WithCount: false,

		MainQuery:  "select q.id from v_test q limit 10 offset 2",
		CountQuery: "",
	},
	{ // 18. Test ERROR empty query
		Target:    "v_test",
		Params:    "",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Request parameters is not passed"),
	},
	{ // 19. Test ERROR empty query
		Target:    "",
		Params:    "??",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Request target not passed"),
	},
	{ // 20. Test ERROR unexpected fieldName in select
		Target:    "v_test",
		Params:    "randomField??",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected field name in select - randomField"),
	},
	{ // 21. Test ERROR unexpected fieldName in condition
		Target:    "v_test",
		Params:    "?randomField==1?",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected field name in condition - randomField"),
	},
	{ // 22. Test ERROR unsupported operator in condition
		Target:    "v_test",
		Params:    "?randomField*=1?",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unsupported operator in condition - randomField"),
	},
	{ // 23. Test ERROR unexpected orderField in rests
		Target:    "v_test",
		Params:    "??ID,dasc,10,0",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection order - dasc"),
	},
	{ // 24. Test ERROR unexpected sort order in rests
		Target:    "v_test",
		Params:    "??UD,desc,10,0",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection order field - UD"),
	},
	{ // 25. Test ERROR unexpected orderField in rests
		Target:    "v_test",
		Params:    "??UD,desc,10,0",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection order field - UD"),
	},
	{ // 26. Test ERROR unexpected limit in rests
		Target:    "v_test",
		Params:    "??ID,desc,a,0",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection limit - a"),
	},
	{ // 27. Test ERROR negative limit in rests
		Target:    "v_test",
		Params:    "??ID,desc,-1,0",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Invaild negative selection limit - -1"),
	},
	{ // 28. Test ERROR unexpected offset in rests
		Target:    "v_test",
		Params:    "??ID,desc,10,a",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection offset - a"),
	},
	{ // 29. Test ERROR negative offset in rests
		Target:    "v_test",
		Params:    "??ID,desc,10,-1",
		WithCount: false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Invaild negative selection offset - -1"),
	},
}

func TestGet(t *testing.T) {
	for index, c := range testGetCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			mainQuery, countQuery, err := Get(TestModel{}, c.Target, c.Params, c.WithCount)
			if err != nil && err.Error() != c.Err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if mainQuery != c.MainQuery {
				t.Errorf("expected: %v, got: %v", c.MainQuery, mainQuery)
				t.Fail()
			}
			if c.WithCount && countQuery != c.CountQuery {
				t.Errorf("expected: %v, got: %v", c.CountQuery, countQuery)
				t.Fail()
			}
		})
	}
}

var testSearchCases = []struct {
	// Search params
	ModelsMap    map[string]map[string]string
	Target       string
	Params       string
	WithCount    bool
	SearchParams string

	// Search response
	MainQuery  string
	CountQuery string
	Err        error
}{
	{ // 1. Test empty query and search blocks
		Target:    "v_test",
		Params:    "??",
		WithCount: true,

		MainQuery:  "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
	},
	{ // 2. Test empty query params block and string params search
		Target:       "v_test",
		Params:       "??",
		WithCount:    true,
		SearchParams: "content~~something",

		MainQuery:  "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q where (lower(q.content::text) like '%something%')",
		CountQuery: "select count(*) from (select 1 from v_test q where (lower(q.content::text) like '%something%')) q",
	},
	{ // 3. Test search query with select block
		Target:       "v_test",
		Params:       "ID,content,count??",
		WithCount:    true,
		SearchParams: "ID~~123",

		MainQuery:  "select q.id, q.content, q.count from v_test q where (lower(q.id::text) like '%123%')",
		CountQuery: "select count(*) from (select 1 from v_test q where (lower(q.id::text) like '%123%')) q",
	},
	{ // 4. Test search query with select and conditions block
		Target:       "v_test",
		Params:       "isBool?ID==1?",
		WithCount:    true,
		SearchParams: "extraField~~ok",

		MainQuery:  "select q.is_bool from v_test q where (lower(q.extra_field::text) like '%ok%') and q.id = 1",
		CountQuery: "select count(*) from (select 1 from v_test q where (lower(q.extra_field::text) like '%ok%') and q.id = 1) q",
	},
	{ // 5. Test search query with complex compilation block
		Target:       "v_test",
		Params:       "ID?(content==something)*extraField!=anything?ID,desc,5,0",
		WithCount:    true,
		SearchParams: "extraField~~ok",

		MainQuery:  "select q.id from v_test q where (lower(q.extra_field::text) like '%ok%') and (q.content = 'something') and q.extra_field != 'anything' order by q.id desc limit 5 offset 0",
		CountQuery: "select count(*) from (select 1 from v_test q where (lower(q.extra_field::text) like '%ok%') and (q.content = 'something') and q.extra_field != 'anything') q",
	},
	{ // 6. Test search query with wrong field name
		Target:       "v_test",
		Params:       "ID??",
		WithCount:    true,
		SearchParams: "extra_Field~~ok",

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected field name in search condition - extra_Field"),
	},
	{ // 7. Test search query with multiple conditions in search block and emtpy main block
		Target:       "v_test",
		Params:       "ID??ID,,,",
		WithCount:    false,
		SearchParams: "extraField~~ok||content~~something||content~~anything",

		MainQuery:  "select q.id from v_test q where (lower(q.extra_field::text) like '%ok%' or lower(q.content::text) like '%something%' or lower(q.content::text) like '%anything%') order by q.id",
		CountQuery: "",
	},
	{ // 8. Test search query with multiple conditions in search block and single main condition
		Target:       "v_test",
		Params:       "ID?isBool==true?",
		WithCount:    false,
		SearchParams: "extraField~~ok||content~~something||content~~anything",

		MainQuery:  "select q.id from v_test q where (lower(q.extra_field::text) like '%ok%' or lower(q.content::text) like '%something%' or lower(q.content::text) like '%anything%') and q.is_bool = true",
		CountQuery: "",
	},
	{ // 9. Test search query with multiple conditions in search block and complex main conditions block
		Target:       "v_test",
		Params:       "ID?isBool==true||content==anything?",
		WithCount:    false,
		SearchParams: "extraField~~any||content~~something||content~~nothing",

		MainQuery:  "select q.id from v_test q where (lower(q.extra_field::text) like '%any%' or lower(q.content::text) like '%something%' or lower(q.content::text) like '%nothing%') and q.is_bool = true or q.content = 'anything'",
		CountQuery: "",
	},
	{ // 10. Test search query with multiple conditions in search block and complex main query
		Target:       "v_test",
		Params:       "ID,isBool?isBool==true||content==anything?ID,desc,10,0",
		WithCount:    false,
		SearchParams: "extraField~~any",

		MainQuery:  "select q.id, q.is_bool from v_test q where (lower(q.extra_field::text) like '%any%') and q.is_bool = true or q.content = 'anything' order by q.id desc limit 10 offset 0",
		CountQuery: "",
	},
	{ // 11. Test search query bracket block
		Target:       "v_test",
		Params:       "ID??",
		WithCount:    false,
		SearchParams: "extraField~~ok||(content~~1&&content~~2)",

		MainQuery:  "select q.id from v_test q where ((lower(q.extra_field::text) like '%ok%' or (lower(q.content::text) like '%1%' and lower(q.content::text) like '%2%'))",
		CountQuery: "",
	},
}

func TestSearch(t *testing.T) {
	for index, c := range testSearchCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			mainQuery, countQuery, err := Search(TestModel{}, c.Target, c.Params, c.WithCount, c.SearchParams)
			if err != nil && err.Error() != c.Err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if mainQuery != c.MainQuery {
				t.Errorf("expected: %v, got: %v", c.MainQuery, mainQuery)
				t.Fail()
			}
			if c.WithCount && countQuery != c.CountQuery {
				t.Errorf("expected: %v, got: %v", c.CountQuery, countQuery)
				t.Fail()
			}
		})
	}
}
