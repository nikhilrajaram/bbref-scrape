package bbrefscrape

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/net/html"
)

// log format helper
func Log(format string, a ...interface{}) {
	log.Println(fmt.Sprintf(format, a...))
}

// log fatal helper
func LogFatal(a ...interface{}) {
	log.Fatal(a...)
}

type IdGenerator struct {
	l sync.Mutex
	i int
}

func NewIdGenerator() IdGenerator {
	return IdGenerator{i: 0}
}

func (ig *IdGenerator) GetId() int {
	ig.l.Lock()
	defer ig.l.Unlock()
	ig.i++
	return ig.i
}

type IdMapper struct {
	l sync.Mutex
	m map[int]string
}

func NewIdMapper() IdMapper {
	return IdMapper{m: make(map[int]string)}
}

func (im *IdMapper) SetName(id int, name string) {
	im.l.Lock()
	defer im.l.Unlock()
	im.m[id] = name
}

func (im *IdMapper) Dump(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		LogFatal(err)
	}
	defer f.Close()

	i, n := 1, len(im.m)
	f.Write([]byte("{\n"))
	for k, v := range im.m {
		f.Write([]byte(fmt.Sprintf("\t\"%d\": \"%s\"", k, v)))
		if i < n {
			f.Write([]byte(",\n"))
		}
		i++
	}
	f.Write([]byte("}"))
}

// dfs search the dom until a terminal condition satisfied
func Search(n *html.Node, isFound func(*html.Node) bool) (*html.Node, bool) {
	if isFound(n) {
		return n, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_n, ok := Search(c, isFound)
		if ok {
			return _n, true
		}
	}
	return nil, false
}

// gets attribute for input html node
// if attribute present in node, returns (attribute, true) tuple
// else, returns ("", false) tuple
func GetAttribute(n *html.Node, key string) (string, bool) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			return n.Attr[i].Val, true
		}
	}

	return "", false
}

// gets inner text for input html node
// if node has inner text, returns (text, true) tuple
// else, returns ("", false) tuple
func GetText(n *html.Node) (string, bool) {
	buf := new(bytes.Buffer)
	html.Render(buf, n)
	z := html.NewTokenizer(buf)

	for {
		switch z.Next() {
		case html.TextToken:
			return string(z.Text()), true
		case html.ErrorToken:
			return "", false
		}
	}
}
