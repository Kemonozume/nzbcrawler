package town

import (
	"io/ioutil"
	"testing"
)

func BuildTestTownParser() *TownParser {
	by, err := ioutil.ReadFile("parser_data")
	if err != nil {
		panic(err)
	}
	tp, _ := NewTownParserWithBytes(by)
	return tp
}

func Test_ParseReleases(t *testing.T) {
	tp := BuildTestTownParser()
	rel := tp.ParseReleases()
	if len(rel) != 5 {
		t.Error("didnt find the right amount of releases, found ", len(rel))
	}
}
