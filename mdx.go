package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

type MdxDict struct {
	// For personal usage, "oald9.json", or "Longman Dictionary of Contemporary English"
	mdxFile string
	// Only match the mdx with the same mdxFile name
	mdxCss  string
	mdxDict map[string]string
}

func (g *MdxDict) Load() error {
	g.mdxDict = loadDecodedMdx(filepath.Join(dataPath, "dicts", g.mdxFile))
	if contents, err := os.ReadFile((filepath.Join(dataPath, "dicts", g.mdxCss))); err == nil {
		g.mdxCss = string(contents)
	} else {
		g.mdxCss = ""
		log.Printf("load dicts[%v] css err: %v", g.mdxFile, err)
	}
	return nil
}

// TODO: support multiple mdx lib at the same time.
var globalDict MdxDict

var gbold = "**"
var gitalic = "*"

func queryMDX(word string, f string) string {
	if f == "html" {
		return globalDict.mdxDict[word]
	}
	fd := strings.NewReader(globalDict.mdxDict[word]) // TODO: find a "close" one when missing?
	return parseMDX(fd, f)
}

func parseMDX(info io.Reader, ft string) string {
	doc, err := html.ParseWithOptions(info, html.ParseOptionEnableScripting(false))
	if err != nil {
		log.Fatal(err)
	}
	return format([]string{f(doc, 0, nil, ft)})
}

// f -> readText
func f(n *html.Node, level int, parent *html.Node, ft string) string {
	var bold, italic string
	if ft == "md" {
		bold, italic = "**", "*"
	}
	// log.Printf("LEVEL[%d %p <- %p] Type: [%#v], DataAtom: [%s], Data: [%#v], Namespace: [%#v], Attr: [%#v]", level, n, parent, n.Type, n.DataAtom, n.Data, n.Namespace, n.Attr)
	if n.Type == html.TextNode {
		// t.Logf("text: [%s] level %d", n.Data, level)
		return compressEmptyLine(n.Data)
	}
	if isElement(n, "div", "") {
		// return "\n" + readS(n) + "\n"
		return "\n"
	}
	// italic
	if isElement(n, "i", "") {
		return renderMD(readS(n, ft), italic)
	}
	// bold
	if isElement(n, "b", "") {
		return renderMD(readS(n, ft), bold)
	}
	if isElement(n, "table", "") {
		return "\n" + readS(n, ft) + "\n"
	}
	if isElement(n, "ex", "") {
		return fmt.Sprintf("> %s <", readS(n, ft))
	}
	if n.Type == html.ElementNode && n.DataAtom.String() == "br" {
		return "\n"
	}
	var s string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s += f(c, level+1, n, ft)
	}
	return s
}

func readS(n *html.Node, ft string) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return compressEmptyLine(n.Data)
	}
	var s string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s += f(c, 0, n, ft)
	}
	return s
}

func renderMD(s string, id string) string {
	return id + s + id
}

func loadDecodedMdx(filePath string) map[string]string {
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %v, %v", filePath, err)
	}

	// Define a map to hold the unmarshaled data
	data := make(map[string]string)

	// Unmarshal the JSON data into the map
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	return data
}