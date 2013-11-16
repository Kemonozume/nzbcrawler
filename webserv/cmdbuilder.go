package webserv

import (
	"os"
	"strings"
)

type CMDBuilder struct {
}

func (c *CMDBuilder) Tokenize(token string) string {
	command := "select * from release where "
	atoken := strings.Split(token, "")
	for {
		pos := c.findEarliest(atoken)
		if pos != -1 {
			if pos == 0 { //special char at the beginning
				command = c.buildCommand(command, atoken[pos])
				atoken = atoken[1:]
			} else { //some tag before special char so get the tag...
				cmd := atoken[0:pos]
				spe := atoken[pos]
				command = c.buildCommand(command, strings.Join(cmd, ""))
				command = c.buildCommand(command, spe)
				atoken = atoken[pos+1:]
			}
		} else { //only tag
			command = c.buildCommand(command, strings.Join(atoken[0:], ""))
			atoken = nil
		}
		if atoken == nil {
			break
		}
	}
	return command
}

func (c *CMDBuilder) buildCommand(command string, consumed string) string {
	if consumed == "" {
		return command
	}
	switch consumed {
	case "(", ")":
		command += " " + consumed + " "
	case "|":
		command += " OR "
	case "&":
		command += " AND "
	default:
		if strings.Contains(consumed, "!") {
			consumed = strings.Replace(consumed, "!", "", -1)
			command += "tag NOT LIKE '%" + consumed + "%'"
		} else {
			command += "tag LIKE '%" + consumed + "%'"
		}
	}
	return command
}

func (c *CMDBuilder) findEarliest(token []string) int {
	for i, val := range token {
		if val == "(" || val == ")" || val == "|" || val == "&" {
			return i
		}
	}
	return -1
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
