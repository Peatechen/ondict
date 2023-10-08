package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/net/html"
)

var dialTimeout = 5 * time.Second
var idleTimeout = 30 * time.Second

var help = flag.Bool("h", false, "show this help doc")
var word = flag.String("q", "", "specify the word that you want to query")
var easyMode = flag.Bool("e", false, "true to show only 'frequent' meaning")
var dev = flag.Bool("d", false, "if specified, a static html file will be parsed, instead of an online query")
var verbose = flag.Bool("v", false, "show debug logs")
var interactive = flag.Bool("i", false, "launch an interactive CLI app")
var server = flag.Bool("s", false, "serve as a HTTP server, for cache stuff, make it quicker!")
var remote = flag.String("c", "", "it can serve as a HTTP client, to get response from server")
var colour = flag.Bool("color", false, "This flags controls whether to use colors.")

var mu sync.Mutex // owns history
var history map[string]string = make(map[string]string)
var dataPath string
var historyFile string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	config := filepath.Join(home, ".config")
	dataPath = filepath.Join(config, "ondict")
	historyFile = filepath.Join(dataPath, "history.json")
	if dataPath == "" || historyFile == "" {
		log.Fatalf("empty datapath/historyfile: %v||%v", dataPath, historyFile)
	}
}

func main() {
	flag.Parse()
	if *help || flag.NFlag() == 0 || len(flag.Args()) > 0 {
		flag.PrintDefaults()
		return
	}
	if !*verbose {
		log.SetOutput(io.Discard)
	}
	if !*colour {
		color.NoColor = true
	}

	if *interactive {
		startLoop()
		return
	}

	if *server {
		stop := make(chan error)
		p := new(proxy)
		p.timeout = time.NewTimer(idleTimeout)
		dp, err := os.Executable()
		if err != nil {
			log.Fatalf("getting ondict path error: %v", err)
		}
		network, addr := autoNetworkAddressPosix(dp, "")
		if _, err := os.Stat(addr); err == nil {
			if err := os.Remove(addr); err != nil {
				log.Fatalf("removing remote socket file: %v", err)
			}
		}
		log.Printf("%s, start a new server: %s", dp, addr)
		l, err := net.Listen(network, addr)
		if err != nil {
			log.Fatal("bad Listen: ", err)
		}
		server := http.Server{
			Handler: p,
		}

		go func() {
			if err := server.Serve(l); err != nil {
				stop <- err
				close(stop)
			}
		}()

		select {
		case c := <-p.timeout.C:
			log.Fatal("timeout, server down!", c)
		case err := <-stop:
			log.Fatal("server down", err)
		}
	}

	if *remote == "auto" {
		dp, err := os.Executable()
		if err != nil {
			log.Fatalf("getting ondict path error: %v", err)
		}
		network, address := autoNetworkAddressPosix(dp, "")
		log.Printf("auto mode dp: %v, network: %v, address: %v", dp, network, address)
		netConn, err := net.DialTimeout(network, address, dialTimeout)

		if err == nil { // detect an exsitng server, just forward a request
			if err := request(netConn); err != nil {
				log.Fatal(err)
			}
			return
		}
		if network == "unix" {
			// Sometimes the socketfile isn't properly cleaned up when the server
			// shuts down. Since we have already tried and failed to dial this
			// address, it should *usually* be safe to remove the socket before
			// binding to the address.
			// TODO(rfindley): there is probably a race here if multiple server
			// instances are simultaneously starting up.
			if _, err := os.Stat(address); err == nil {
				if err := os.Remove(address); err != nil {
					log.Fatalf("removing remote socket file: %v", err)
				}
			}
		}
		args := []string{
			"-s=true",
		}
		log.Printf("starting remote: %v", args)
		if err := startRemote(dp, args...); err != nil {
			log.Fatal(err)
		}
		const retries = 5
		// It can take some time for the newly started server to bind to our address,
		// so we retry for a bit.
		for retry := 0; retry < retries; retry++ {
			startDial := time.Now()
			netConn, err = net.DialTimeout(network, address, dialTimeout)
			if err == nil {
				if err := request(netConn); err != nil {
					log.Fatal(err)
				}
				return
			}
			log.Printf("failed attempt #%d to connect to remote: %v\n", retry+2, err)
			// In case our failure was a fast-failure, ensure we wait at least
			// f.dialTimeout before trying again.
			if retry != retries-1 {
				time.Sleep(dialTimeout - time.Since(startDial))
			}
		}
		os.Exit(3)
	}

	// just for offline test.
	if *dev {
		fd, err := os.Open("./doctor_ldoce.html")
		if err != nil {
			log.Fatal(err)
		}
		defer fd.Close()
		fmt.Println(parseHTML(fd))
		return
	}
	fmt.Println(queryByURL(*word))
}

func query(word string) string {
	var res string
	mu.Lock()
	if ex, ok := history[word]; ok {
		log.Printf("cache hit!")
		res = ex
	} else {
		res = queryByURL(word)
		history[word] = res
	}
	mu.Unlock() // TODO: performance
	return res
}

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

func pureEmpty(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\n' || c == '\t' || c == '\u00a0' {
			continue
		}
		return false
	}
	return true
}

func format(input []string) string {
	// TODO: remove consecutive CRLFs or "empty lines"?
	return strings.Join(input, "\n")
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
		return []string{blue("%s", fmt.Sprintf("frequent HEAD %s", readText(n)))}
	}
	// read Sense for DEF
	if isElement(n, "span", "Sense") {
		red := color.New(color.FgRed).SprintfFunc()
		return []string{red("%s", fmt.Sprintf("Sense(%v):%s", getSpanID(n), readText(n)))}
	}
	if isElement(n, "span", "Head") {
		cyan := color.New(color.FgCyan).SprintfFunc()
		return []string{cyan("%s", fmt.Sprintf("HEAD %s", readText(n)))}
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
		res = append(res, fmt.Sprintf("==find an ldoce entry=="))
		res = append(res, readLongmanEntry(n)...)
		return res
	}

	if !*easyMode && isElement(n, "span", "bussdictEntry Entry") {
		res = append(res, fmt.Sprintf("==find a buss entry=="))
		res = append(res, readLongmanEntry(n)...)
		return readLongmanEntry(n)
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
		return n.Data
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
		return noColor("%s", fmt.Sprintf("\n%sEXAMPLE:%s", strings.Repeat(" ", 0), readOneExample(n)))
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

func Restore() {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		log.Printf("open file history err: %v", err)
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &history)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("history: %v", history)
}

func Store() {
	his, err := json.Marshal(history)
	if err != nil {
		log.Fatal("marshal err ", err)
	}
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		log.Fatal("make dir err", err)
	}
	f, err := os.Create(historyFile)
	if err != nil {
		log.Fatal("create file err", err)
	}

	defer f.Close()

	_, err = f.Write(his)

	if err != nil {
		log.Fatal("write file err", err)
	}
}
