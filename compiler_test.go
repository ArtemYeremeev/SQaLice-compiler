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
	WithArgs  bool

	// Get response
	MainQuery  string
	CountQuery string
	Args       []interface{}
	Err        error
}{
	{ // 1. Test empty params blocks
		Target:    "v_test",
		Params:    "??",
		WithCount: true,
		WithArgs:  false,

		MainQuery:  "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
		Err:        newError(""),
	},
	{ // 2. Test fields params block with 1 field
		Target:    "v_test",
		Params:    "ID??",
		WithCount: true,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
		Err:        newError(""),
	},
	{ // 3. Test fields params block with 3 fields
		Target:    "v_test",
		Params:    "ID,content,count??",
		WithCount: true,
		WithArgs:  false,

		MainQuery:  "select q.id, q.content, q.count from v_test q",
		CountQuery: "select count(*) from (select 1 from v_test q) q",
		Err:        newError(""),
	},
	{ // 4. Test conditions params block with 1 condition
		Target:    "v_test",
		Params:    "ID?ID==1?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.id = 1",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 5. Test conditions params block with 1 non-bracket conditionsSet
		Target:    "v_test",
		Params:    "content?ID==1*ID==3?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.content from v_test q where q.id = 1 and q.id = 3",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 6. Test conditions params block with 1 bracket conditionsSet
		Target:    "v_test",
		Params:    "ID?(ID==1||ID==3)?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where (q.id = 1 or q.id = 3)",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 7. Test conditions params block with 2 bracket conditionsSets
		Target:    "v_test",
		Params:    "ID?(ID>1||ID<=3)*(content!=test1*content==test2)?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where (q.id > 1 or q.id <= 3) and (q.content != test1 and q.content = test2)",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 8. Test conditions params block with 1 bracket conditionsSet and 1 non-bracket conditionsSet
		Target:    "v_test",
		Params:    "ID,count?(count!=1*count!=3)*ID==2?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id, q.count from v_test q where (q.count != 1 and q.count != 3) and q.id = 2",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 9. Test conditions params block with 2 bracket conditionsSet and 1 non-bracket conditionsSet
		Target:    "v_test",
		Params:    "content,count?(count!=1*count!=3||ID>=11)||(content==something awful||content==critical404)*ID!=42?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.content, q.count from v_test q where (q.count != 1 and q.count != 3 or q.id >= 11) or (q.content = somethingawful or q.content = critical404) and q.id != 42",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 10. Test conditions params block with 1 non-bracket array condition
		Target:    "v_test",
		Params:    "ID,isBool?ID==1,2,test1?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id, q.is_bool from v_test q where q.id = any(1,2,test1)",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 11. Test conditions params block with 1 non-bracket array conditionsSet
		Target:    "v_test",
		Params:    "count?ID==1,2,test1*content==test2,true?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.count from v_test q where q.id = any(1,2,test1) and q.content = any(test2,true)",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 12 Test conditions params block with OVERLAPS operator and single value
		Target:    "v_test",
		Params:    "ID?content>>value?",
		WithCount: true,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content && value",
		CountQuery: "select count(*) from (select 1 from v_test q where q.content && value) q",
		Err:        newError(""),
	},
	{ // 13 Test conditions params block with OVERLAPS operator and muliple values
		Target:    "v_test",
		Params:    "ID,count?content>>value1,value2,true,14*ID==25?,,10,0",
		WithCount: true,
		WithArgs:  false,

		MainQuery:  "select q.id, q.count from v_test q where q.content && value1,value2,true,14 and q.id = 25 limit 10 offset 0",
		CountQuery: "select count(*) from (select 1 from v_test q where q.content && value1,value2,true,14 and q.id = 25) q",
		Err:        newError(""),
	},
	{ // 14. Test restrictions params block with all restrictions
		Target:    "v_test",
		Params:    "ID?isBool==true?ID,desc,10,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.is_bool = true order by q.id desc limit 10 offset 0",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 15. Test restrictions params block with order field
		Target:    "v_test",
		Params:    "ID??ID,,,",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q order by q.id asc",
		CountQuery: "select count(*) from (select 1 from v_test q order by q.id) q",
		Err:        newError(""),
	},
	{ // 16. Test restrictions params block with limit
		Target:    "v_test",
		Params:    "ID,isBool,content?ID!=42?,,5,",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id, q.is_bool, q.content from v_test q where q.id != 42 limit 5",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 17. Test restrictions params block with limit and offset
		Target:    "v_test",
		Params:    "ID??,,10,2",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q limit 10 offset 2",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 18. Test ERROR empty query
		Target:    "v_test",
		Params:    "",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Request parameters is not passed"),
	},
	{ // 19. Test ERROR empty query
		Target:    "",
		Params:    "??",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Request target not passed"),
	},
	{ // 20. Test ERROR unexpected fieldName in select
		Target:    "v_test",
		Params:    "randomField??",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected field name in select - randomField"),
	},
	{ // 21. Test ERROR unexpected fieldName in condition
		Target:    "v_test",
		Params:    "?randomField==1?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected field name in condition - randomField"),
	},
	{ // 22. Test ERROR unsupported operator in condition
		Target:    "v_test",
		Params:    "?randomField*=1?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unsupported operator in condition - randomField"),
	},
	{ // 23. Test ERROR unexpected orderField in rests
		Target:    "v_test",
		Params:    "??ID,dasc,10,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection order - dasc"),
	},
	{ // 24. Test ERROR unexpected sort order in rests
		Target:    "v_test",
		Params:    "??UD,desc,10,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection order field - UD"),
	},
	{ // 25. Test ERROR unexpected orderField in rests
		Target:    "v_test",
		Params:    "??UD,desc,10,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection order field - UD"),
	},
	{ // 26. Test ERROR unexpected limit in rests
		Target:    "v_test",
		Params:    "??ID,desc,a,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection limit - a"),
	},
	{ // 27. Test ERROR negative limit in rests
		Target:    "v_test",
		Params:    "??ID,desc,-1,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Invaild negative selection limit - -1"),
	},
	{ // 28. Test ERROR unexpected offset in rests
		Target:    "v_test",
		Params:    "??ID,desc,10,a",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Unexpected selection offset - a"),
	},
	{ // 29. Test ERROR negative offset in rests
		Target:    "v_test",
		Params:    "??ID,desc,10,-1",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Invaild negative selection offset - -1"),
	},
	{ // 30. Test array condition with NOT EQUALS operator
		Target:    "v_test",
		Params:    "ID?ID!=test1,test2?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where not q.id = any(test1,test2)",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 31. Test array condition with unexpected operator
		Target:    "v_test",
		Params:    "content?ID<=test1,test2?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected operator in array condition - <="),
	},
	{ // 32. Test nested object condition with integer field
		Target:    "v_test",
		Params:    "ID?content==ID^^1?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content->>'ID' = 1",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 33. Test nested object condition with string field
		Target:    "v_test",
		Params:    "ID?content==content^^test?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content->>'content' = test",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 34. Test nested object condition with array value
		Target:    "v_test",
		Params:    "ID?content==ID^^vla,2?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content->>'ID' = any(vla,2)",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 35. Test condition with IS NULL value
		Target:    "v_test",
		Params:    "ID?content==null?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content is null",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 36. Test multiple conditions with null values
		Target:    "v_test",
		Params:    "ID?content!=null||ID==NULL?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content is not null or q.id is null",
		CountQuery: "",
		Err:        newError(""),
	},
	{ // 37. Test condition with unexpected OVERLAPS operator for NULL value
		Target:    "v_test",
		Params:    "ID?content>>null?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected OVERLAPS operator in NULL condition"),
	},
	{ // 38. Test condition with different unexpected operator for NULL value
		Target:    "v_test",
		Params:    "?content<null?",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "",
		CountQuery: "",
		Err:        newError("Passed unexpected operator in NULL condition - <"),
	},
	{ // 39. Test query with empty condition block (withArgs)
		Target:    "v_test",
		Params:    "??",
		WithCount: false,
		WithArgs:  true,

		MainQuery: "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q",
		Err:        newError(""),
	},
	{ // 40. Test query with single condition (withArgs)
		Target:    "v_test",
		Params:    "?ID==1?",
		WithCount: true,
		WithArgs:  true,

		MainQuery:  "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q where q.id = $1",
		CountQuery: "select count(*) from (select 1 from v_test q where q.id = $1) q",
		Args:       []interface{}{1},
		Err:        newError(""),
	},
	{ // 41. Test query with multiple conditions (withArgs)
		Target:    "v_test",
		Params:    "ID,extraField?ID==1||ID==2||content!=smth?",
		WithCount: true,
		WithArgs:  true,

		MainQuery:  "select q.id, q.extra_field from v_test q where q.id = $1 or q.id = $2 or q.content != $3",
		CountQuery: "select count(*) from (select 1 from v_test q where q.id = $1 or q.id = $2 or q.content != $3) q",
		Args:       []interface{}{1, 2, "smth"},
		Err:        newError(""),
	},
	{ // 42. Test query with bracket and non-bracket conditions (withArgs)
		Target:    "v_test",
		Params:    "ID?(ID==1*content==anth)||content!=smth*ID!=8?",
		WithCount: true,
		WithArgs:  true,

		MainQuery:  "select q.id from v_test q where (q.id = $1 and q.content = $2) or q.content != $3 and q.id != $4",
		CountQuery: "select count(*) from (select 1 from v_test q where (q.id = $1 and q.content = $2) or q.content != $3 and q.id != $4) q",
		Args:       []interface{}{1, "anth", "smth", 8},
		Err:        newError(""),
	},
	{ // 43. Test query with array conditions (withArgs)
		Target:     "v_test",
		Params:     "ID?(ID==1,2,3||content!=new)*isBool==true?ID,desc,10,0",
		WithCount:  true,
		WithArgs:   true,

		MainQuery:  "select q.id from v_test q where (q.id = any($1) or q.content != $2) and q.is_bool = $3 order by q.id desc limit 10 offset 0",
		CountQuery: "select count(*) from (select 1 from v_test q where (q.id = any($1) or q.content != $2) and q.is_bool = $3) q",
		Args:       []interface{}{[]int{1, 2, 3}, "new", true},
		Err:        newError(""),
	},
	{ // 44. Test query with 2 bracket conditions (withArgs)
		Target:     "v_test",
		Params:     "isBool?(ID==null||content!=new)*(isBool==true)?ID,desc,10,0",
		WithCount:  false,
		WithArgs:   true,

		MainQuery:  "select q.is_bool from v_test q where (q.id is null or q.content != $1) and (q.is_bool = $2) order by q.id desc limit 10 offset 0",
		Args:       []interface{}{nil, "new", true},
		Err:        newError(""),
	},
	{ // 45. Test query with nested condition (withArgs)
		Target:     "v_test",
		Params:     "ID?content==content^^smth?",
		WithCount:  false,
		WithArgs:   true,

		MainQuery:  "select q.id from v_test q where q.content->>'content' = $1",
		Args:       []interface{}{"smth"},
		Err:        newError(""),
	},
	{ // 46. Test query with long value in condition
		Target:     "v_test",
		Params:     "?content==content1content2content3content4content5content6content7?",
		WithCount:  false,
		WithArgs:   false,

		MainQuery:  "",
		Err:        newError("Too long string value in condition - content1content2content3content4content5content6content7"),
	},
	{ // 47. Test restrictions params block with only field parameter
		Target:    "v_test",
		Params:    "ID??ID,,,",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q order by q.id asc",
		Err:        newError(""),
	},
	{ // 48. Test restrictions params block with two order fields without order
		Target:    "v_test",
		Params:    "ID?content==smth?isBool|ID,,,",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q where q.content = smth order by q.is_bool asc, q.id asc",
		Err:        newError(""),
	},
	{ // 49. Test restrictions params block with two order fields and other order
		Target:    "v_test",
		Params:    "ID??ID|isBool,desc,10,0",
		WithCount: false,
		WithArgs:  false,

		MainQuery:  "select q.id from v_test q order by q.id desc, q.is_bool desc limit 10 offset 0",
		Err:        newError(""),
	},
}

func TestGet(t *testing.T) {
	for index, c := range testGetCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			mainQuery, countQuery, args, err := Get(TestModel{}, c.Target, c.Params, c.WithCount, c.WithArgs)
			if err != nil && err.Error() != c.Err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if mainQuery != c.MainQuery {
				t.Errorf("expected mainQ: %v, got: %v", c.MainQuery, mainQuery)
				t.Fail()
			}
			if c.WithCount && countQuery != c.CountQuery {
				t.Errorf("expected countQ: %v, got: %v", c.CountQuery, countQuery)
				t.Fail()
			}

			if c.WithArgs {
				for i, v := range args {
					_, ok := c.Args[i].([]int)
					if ok {
						continue
					} else if c.Args[i] != v {
						t.Errorf("expected arg: %v, got: %v", c.Args[i], v)
						t.Fail()
					}
				}
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
	WithArgs     bool
	SearchParams string

	// Search response
	MainQuery  string
	CountQuery string
	Args       []interface{}
	Err        error
}{
	/* { // 1. Test empty query and search blocks
		Target:       "v_test",
		Params:       "??",
		WithCount:    true,
		WithArgs:     false,

		MainQuery:    "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q",
		CountQuery:   "select count(*) from (select 1 from v_test q) q",
		Err:          newError(""),
	},
	{ // 2. Test empty query params block and string params search
		Target:       "v_test",
		Params:       "??",
		WithCount:    true,
		WithArgs:     false,
		SearchParams: "content~~something",

		MainQuery:    "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q where (lower(q.content::text) like %something%)",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.content::text) like %something%)) q",
		Err:          newError(""),
	},
	{ // 3. Test search query with select block
		Target:       "v_test",
		Params:       "ID,content,count??",
		WithCount:    true,
		WithArgs:     false,
		SearchParams: "ID~~123",

		MainQuery:    "select q.id, q.content, q.count from v_test q where (lower(q.id::text) like %123%)",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.id::text) like %123%)) q",
		Err:          newError(""),
	},
	{ // 4. Test search query with select and conditions block
		Target:       "v_test",
		Params:       "isBool?ID==1?",
		WithCount:    true,
		WithArgs:     false,
		SearchParams: "extraField~~ok",

		MainQuery:    "select q.is_bool from v_test q where (lower(q.extra_field::text) like %ok%) and q.id = 1",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.extra_field::text) like %ok%) and q.id = 1) q",
		Err:          newError(""),
	},
	{ // 5. Test search query with complex compilation block
		Target:       "v_test",
		Params:       "ID?(content==something)*extraField!=anything?ID,desc,5,0",
		WithCount:    true,
		WithArgs:     false,
		SearchParams: "extraField~~ok",

		MainQuery:    "select q.id from v_test q where (lower(q.extra_field::text) like %ok%) and (q.content = something) and q.extra_field != anything order by q.id desc limit 5 offset 0",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.extra_field::text) like %ok%) and (q.content = something) and q.extra_field != anything) q",
		Err:          newError(""),
	},
	{ // 6. Test search query with wrong field name
		Target:       "v_test",
		Params:       "ID??",
		WithCount:    true,
		WithArgs:     false,
		SearchParams: "extra_Field~~ok",

		MainQuery:    "",
		CountQuery:   "",
		Err:          newError("Passed unexpected field name in search condition - extra_Field"),
	},
	{ // 7. Test search query with multiple conditions in search block and emtpy main block
		Target:       "v_test",
		Params:       "ID??ID,,,",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "extraField~~ok||content~~something||content~~anything",

		MainQuery:    "select q.id from v_test q where (lower(q.extra_field::text) like %ok% or lower(q.content::text) like %something% or lower(q.content::text) like %anything%) order by q.id asc",
		CountQuery:   "",
		Err:          newError(""),
	},
	{ // 8. Test search query with multiple conditions in search block and single main condition
		Target:       "v_test",
		Params:       "ID?isBool==true?",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "extraField~~ok||content~~something||content~~anything",

		MainQuery:    "select q.id from v_test q where (lower(q.extra_field::text) like %ok% or lower(q.content::text) like %something% or lower(q.content::text) like %anything%) and q.is_bool = true",
		CountQuery:   "",
		Err:          newError(""),
	},
	{ // 9. Test search query with multiple conditions in search block and complex main conditions block
		Target:       "v_test",
		Params:       "ID?isBool==true||content==anything?",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "extraField~~any||content~~something||content~~nothing",

		MainQuery:    "select q.id from v_test q where (lower(q.extra_field::text) like %any% or lower(q.content::text) like %something% or lower(q.content::text) like %nothing%) and q.is_bool = true or q.content = anything",
		CountQuery:   "",
		Err:          newError(""),
	},
	{ // 10. Test search query with multiple conditions in search block and complex main query
		Target:       "v_test",
		Params:       "ID,isBool?isBool==true||content==anything?ID,desc,10,0",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "extraField~~any",

		MainQuery:    "select q.id, q.is_bool from v_test q where (lower(q.extra_field::text) like %any%) and q.is_bool = true or q.content = anything order by q.id desc limit 10 offset 0",
		CountQuery:   "",
		Err:          newError(""),
	},
	{ // 11. Test search query bracket block
		Target:       "v_test",
		Params:       "ID??",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "(content~~1||content~~2)*extraField~~ok",

		MainQuery:    "select q.id from v_test q where ((lower(q.content::text) like %1% or lower(q.content::text) like %2%) and lower(q.extra_field::text) like %ok%)",
		CountQuery:   "",
		Err:          newError(""),
	}, */
	{ // 12. Test searchQuery with two bracket blocks
		Target:       "v_test",
		Params:       "ID,content,extraField??ID,desc,,",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "(content~~1||content~~2)*(extraField~~some||extraField~~any)",

		MainQuery:    "select q.id, q.content, q.extra_field from v_test q where ((lower(q.content::text) like %1% or lower(q.content::text) like %2%) and (lower(q.extra_field::text) like %some% or lower(q.extra_field::text) like %any%)) order by q.id desc",
		CountQuery:   "",
		Err:          newError(""),
	},
	{ // 13. Test nested object condition search
		Target:       "v_test",
		Params:       "ID??",
		WithCount:    false,
		WithArgs:     false,
		SearchParams: "content~~content^^ая/n",

		MainQuery:    "select q.id from v_test q where (lower(q.content->>'content'::text) like %ая%n%)",
		CountQuery:   "",
		Err:          newError(""),
	},
	{ // 14. Test single search condition (withArgs)
		Target:       "v_test",
		Params:       "??",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "content~~something",

		MainQuery:    "select q.id, q.content, q.count, q.extra_field, q.is_bool, q.one_more_field from v_test q where (lower(q.content::text) like $1)",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.content::text) like $1)) q",
		Args:         []interface{}{"%something%"},
		Err:          newError(""),
	},
	{ // 15. Test one search + one get conditions query (withArgs)
		Target:       "v_test",
		Params:       "content?ID!=1?ID,asc,,",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "ID~~1",

		MainQuery:    "select q.content from v_test q where (lower(q.id::text) like $1) and q.id != $2 order by q.id asc",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.id::text) like $1) and q.id != $2) q",
		Args:         []interface{}{"%1%", 1},
		Err:          newError(""),
	},
	{ // 16. Test multiple bracket and non-bracket conditions (withArgs)
		Target:       "v_test",
		Params:       "ID?(ID==1,2,13||isBool==true)*content!=anth?",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "ID~~1||content~~smth",

		MainQuery:    "select q.id from v_test q where (lower(q.id::text) like $1 or lower(q.content::text) like $2) and (q.id = any($3) or q.is_bool = $4) and q.content != $5",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.id::text) like $1 or lower(q.content::text) like $2) and (q.id = any($3) or q.is_bool = $4) and q.content != $5) q",
		Args:         []interface{}{"%1%", "%smth%", []int{1, 2, 13}, true, "anth"},
		Err:          newError(""),
	},
	{ // 17. Test overlaps standart confition (withArgs)
		Target:       "v_test",
		Params:       "content?content>>1,2?",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "content~~smth",

		MainQuery:    "select q.content from v_test q where (lower(q.content::text) like $1) and q.content && $2",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.content::text) like $1) and q.content && $2) q",
		Args:         []interface{}{"%smth%", []int{1,2}},
		Err:          newError(""),
	},
	{ // 18. Test search with multiple order fields
		Target:       "v_test",
		Params:       "content?ID!=1?isBool|ID,,30,",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "content~~smth",

		MainQuery:    "select q.content from v_test q where (lower(q.content::text) like $1) and q.id != $2 order by q.is_bool asc, q.id asc limit 30",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.content::text) like $1) and q.id != $2) q",
		Args:         []interface{}{"%smth%", 1},
		Err:          newError(""),
	},
	{ // 19. Test search without limit and order fields
		Target:       "v_test",
		Params:       "content?ID!=1?,desc,,5",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "content~~smth",

		MainQuery:    "select q.content from v_test q where (lower(q.content::text) like $1) and q.id != $2 offset 5",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.content::text) like $1) and q.id != $2) q",
		Args:         []interface{}{"%smth%", 1},
		Err:          newError(""),
	},
	{ // 20. Test search condition with brackets in search string
		Target:       "v_test",
		Params:       "ID??",
		WithCount:    true,
		WithArgs:     true,
		SearchParams: "content~~(anth)",

		MainQuery:    "select q.id from v_test q where (lower(q.content::text) like $1)",
		CountQuery:   "select count(*) from (select 1 from v_test q where (lower(q.content::text) like $1)) q",
		Args:         []interface{}{"%%anth%%", 1},
		Err:          newError(""),
	},
}

func TestSearch(t *testing.T) {
	for index, c := range testSearchCases {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			mainQuery, countQuery, args, err := Search(TestModel{}, c.Target, c.Params, c.WithCount, c.WithArgs, c.SearchParams)
			if err != nil && err.Error() != c.Err.Error() {
				t.Errorf("expected err: %v, got: %v", c.Err, err)
				t.FailNow()
			}

			if mainQuery != c.MainQuery {
				t.Errorf("expected mainQ: %v, got: %v", c.MainQuery, mainQuery)
				t.Fail()
			}
			if c.WithCount && countQuery != c.CountQuery {
				t.Errorf("expected countQ: %v, got: %v", c.CountQuery, countQuery)
				t.Fail()
			}

			if c.WithArgs {
				for i, v := range args {
					_, ok := c.Args[i].([]int)
					if ok {
						continue
					} else if c.Args[i] != v {
						t.Errorf("expected arg: %v, got: %v", c.Args[i], v)
						t.Fail()
					}
				}
			}
		})
	}
}
