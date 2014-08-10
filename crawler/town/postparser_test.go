package town

import (
	"io/ioutil"
	"testing"
)

func BuildTest() *TownPostParser {
	by, err := ioutil.ReadFile("postparser_data")
	if err != nil {
		panic(err)
	}
	tp, _ := NewTownPostParserWithBytes(by)
	return tp
}

func Test_ParsePostId(t *testing.T) {
	tp := BuildTest()
	if id, err := tp.GetPostId(); err != nil {
		t.Error(err.Error())
	} else {
		if id == "759925" {
			t.Log("postid is correct")
		} else {
			t.Error("postid should be 759925 but was ", id)
		}
	}
}

func Test_ParseSecurityToken(t *testing.T) {
	tp := BuildTest()
	if id, err := tp.GetSecurityToken(); err != nil {
		t.Error(err.Error())
	} else {
		if id == "1406214109-e187e49057c63a380d0013fa02fe110953d4c71a" {
			t.Log("postid is correct")
		} else {
			t.Error("security token should be 1406214109-e187e49057c63a380d0013fa02fe110953d4c71a but was ", id)
		}
	}
}
