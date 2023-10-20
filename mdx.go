package main

import (
	"fmt"
	"io"
	"log"

	"golang.org/x/net/html"
)

var bold = "**"
var italic = "*"

func parseMDX(info io.Reader) string {
	doc, err := html.ParseWithOptions(info, html.ParseOptionEnableScripting(false))
	if err != nil {
		log.Fatal(err)
	}
	return format([]string{f(doc, 0, nil)})
}

// f -> readText
func f(n *html.Node, level int, parent *html.Node) string {
	// log.Printf("LEVEL[%d %p <- %p] Type: [%#v], DataAtom: [%s], Data: [%#v], Namespace: [%#v], Attr: [%#v]", level, n, parent, n.Type, n.DataAtom, n.Data, n.Namespace, n.Attr)
	if level == 0 {
		// t.Logf("ROOT: type: %v atom[%s]", n.Type, n.DataAtom)
	}
	if n.Type == html.TextNode {
		// t.Logf("text: [%s] level %d", n.Data, level)
		return compressEmptyLine(n.Data)
	}
	if isElement(n, "div", "") {
		// TODO: origin?
		// return "\n" + readS(n) + "\n"
		return "\n"
	}
	// italic
	if isElement(n, "i", "") {
		return renderMD(readS(n), italic)
	}
	// bold
	if isElement(n, "b", "") {
		return renderMD(readS(n), bold)
	}
	if isElement(n, "table", "") {
		return "\n" + readS(n) + "\n"
	}
	if isElement(n, "ex", "") {
		return fmt.Sprintf("EXAMPLE:%s", readS(n))
	}
	if n.Type == html.ElementNode && n.DataAtom.String() == "br" {
		return "\n"
	}
	var s string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s += f(c, level+1, n)
	}
	return s
}

func readS(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return compressEmptyLine(n.Data)
	}
	var s string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s += f(c, 0, n)
	}
	return s
}

func renderMD(s string, id string) string {
	return id + s + id
}
