package main

import "io/ioutil"

const (
	portalFilename = "~/.cache/o/portal.txt" // TODO: Use XDG_CACHE_HOME
)

type Portal struct {
	absFilename string
	lineNumber  LineNumber
}

func (p *Portal) Save() error {
	s := p.absFilename + "\n" + p.lineNumber.String() + "\n"
	return ioutil.WriteFile(expandUser(portalFilename), []byte(s), 0600)
}
