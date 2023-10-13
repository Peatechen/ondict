package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
	"golang.org/x/net/html"
)

func queryByURL(word string) string {
	start := time.Now()
	// url := fmt.Sprintf("https://ldoceonline.com/dictionary/%s", word)
	url := fmt.Sprintf("https://ldoceonline.com/search/english/direct/?q=%s", url.QueryEscape(word))
	resp, err := http.Get(url)
	log.Printf("query %q cost: %v", url, time.Since(start))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	return parseHTML(resp.Body)
}

func parseHTML(info io.Reader) string {
	doc, err := html.ParseWithOptions(info, html.ParseOptionEnableScripting(false))
	if err != nil {
		log.Fatal(err)
	}
	// Type      NodeType
	// DataAtom  atom.Atom
	// Data      string
	// Namespace string
	// Attr      []Attribute
	var res []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		// log.Printf("Type: [%#v], DataAtom: [%s], Data: [%#v], Namespace: [%#v], Attr: [%#v]", n.Type, n.DataAtom, n.Data, n.Namespace, n.Attr)
		if isElement(n, "div", "dictionary") {
			res = ldoceDict(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	// log.Printf("result: %v", readText(doc))
	f(doc)
	return format(res)
}

func compressEmptyLine(s string) string {
	if pureEmptyLineEndLF(s) {
		return "\n"
	}
	return s
}

// pureEmptyLine returns whether it's an empty line, only consisting of "IsSpace" Code
func pureEmptyLineLF(s string) bool {
	lf := false
	for _, c := range s {
		if c == '\n' || c == '\u0020' {
			lf = true
		}
		if !unicode.IsSpace(c) {
			return false
		}
	}
	return lf && true
}

// pureEmptyLineLF returns whether it's an empty line ended with '\n' or '\u00a0'
func pureEmptyLineEndLF(s string) bool {
	var last rune
	for _, c := range s {
		last = c
		if unicode.IsSpace(c) {
			continue
		}
		return false
	}
	return last == '\n' || last == '\u00a0'
}

// format does:
// 1. compress a ["\n                "] + ["\u00a0"] sequence into one "\n"
// 2. remove consecutive CRLFs(the input lines are has been "compressed" in readText)
// TODO: make it elegant and robust.
func format(input []string) string {
	tmp := make([]string, 0, len(input))
	for i, s := range input {
		if i < len(input)-1 && input[i] == "\n                " && input[i+1] == "\u00a0" {
			continue
		}
		tmp = append(tmp, s)
	}
	joined := strings.Join(tmp, "\n")
	var res string
	lf := false
	for _, c := range joined {
		if c == '\n' || c == '\u00a0' || c == ' ' {
			if lf {
				continue
			}
			lf = true
		} else {
			lf = false
		}
		res += string(c)
	}
	return res
}

func findFirstSubSpan(n *html.Node, class string) *html.Node {
	log.Printf("find class: %q, Type: [%#v], DataAtom: [%s], Data: [%#v], Namespace: [%#v], Attr: [%#v]", class, n.Type, n.DataAtom, n.Data, n.Namespace, n.Attr)
	if isElement(n, "span", class) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := findFirstSubSpan(c, class); res != nil {
			return res
		}
	}
	return nil
}

func readLongmanEntry(n *html.Node) []string {
	// read "frequent head" for PRON
	if isElement(n, "span", "frequent Head") {
		blue := color.New(color.FgBlue).SprintfFunc()
		return []string{blue("%s", fmt.Sprintf("%s", readText(n)))}
	}
	// read Sense for DEF
	if isElement(n, "span", "Sense") {
		red := color.New(color.FgRed).SprintfFunc()
		sense := fmt.Sprintf("%sDEF:%s", strings.Repeat("\t", 1), readText(n))
		log.Printf("Sense: %q", sense)
		return []string{red("%s", sense)}
	}
	if isElement(n, "span", "Head") {
		cyan := color.New(color.FgCyan).SprintfFunc()
		return []string{cyan("%s", fmt.Sprintf("%s", readText(n)))}
	}
	var res []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res = append(res, readLongmanEntry(c)...)
	}
	return res
}

func ldoceDict(n *html.Node) []string {
	var res []string
	if isElement(n, "span", "ldoceEntry Entry") {
		res = append(res, fmt.Sprintf("\n*****LDOCE ENTRY*****\n"))
		res = append(res, readLongmanEntry(n)...)
		return res
	}

	if !*easyMode && isElement(n, "span", "bussdictEntry Entry") {
		res = append(res, fmt.Sprintf("\n*****BUSS ENTRY*****\n"))
		res = append(res, readLongmanEntry(n)...)
		return res
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res = append(res, ldoceDict(c)...)
	}

	return res
}

func isElement(n *html.Node, ele string, class string) bool {
	if n.Type == html.ElementNode && n.DataAtom.String() == ele {
		if class == "" {
			return true
		}
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == class {
				log.Printf("[wft] readElement good %v, %v, %#v", ele, class, n.Data)
				return true
			}
		}
	}
	return false
}

func readOneExample(n *html.Node) string {
	var s string
	defer func() {
		log.Printf("example[%q]:", s)
	}()
	if n.Type == html.TextNode {
		return n.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s += readText(c)
	}
	return s
}

func readText(n *html.Node) string {
	if n.Type == html.TextNode {
		log.Printf("text: [%q]", n.Data)
		return compressEmptyLine(n.Data)
	}
	if isElement(n, "script", "") {
		return ""
	}
	if getSpanClass(n) == "HWD" {
		return ""
	}
	if getSpanClass(n) == "FIELD" {
		return ""
	}
	if getSpanClass(n) == "ACTIV" {
		return ""
	}
	if isElement(n, "span", "EXAMPLE") {
		noColor := color.New().SprintfFunc()
		return noColor("%s", fmt.Sprintf("\n\u00a0%sEXAMPLE:> %s <\n", strings.Repeat("\t", 2), strings.TrimLeft(readOneExample(n), " \n")))
	}
	var s string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s += readText(c)
	}
	return s
}

func getSpanID(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom.String() == "span" {
		for _, a := range n.Attr {
			if a.Key == "id" {
				return a.Val
			}
		}
	}
	return ""
}

func getSpanClass(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom.String() == "span" {
		for _, a := range n.Attr {
			if a.Key == "class" {
				return a.Val
			}
		}
	}
	return ""
}
