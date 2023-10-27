package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type proxy struct {
	timeout *time.Timer
}

func (s *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.timeout.Stop() {
		select {
		case t := <-s.timeout.C: // try to drain from the channel
			log.Printf("drained from timer: %v", t)
		default:
		}
	}
	if *idleTimeout > 0 {
		s.timeout.Reset(*idleTimeout)
	}
	log.Printf("query HTTP path: %v", r.URL.Path)
	if r.URL.Path == "/dict" {
		q := r.URL.Query()
		word := q.Get("query")
		e := q.Get("engine")
		f := q.Get("format")
		log.Printf("query dict: %v, engine: %v, format: %v", word, e, f)

		res := query(word, e, f)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(res))
		w.Write([]byte("<style>" + oald9css + "</style>"))
		return
	}
	http.FileServer(http.Dir(".")).ServeHTTP(w, r)
}

const oald9css = `@font-face{font-family:'oalecd9';src:url("oalecd9.ttf");font-weight:400;font-style:normal}
body{background-color:#fffefe;font-family:'oalecd9';counter-reset:sn_blk_counter}
.cixing_part{counter-reset:sn_blk_counter}
.cixing_tiaozhuan_part{display:inline;color:#c70000}
.cixing_tiaozhuan_part a:link{text-decoration:none;font-weight:600}
.cixing_tiaozhuan_part a{color:#c70000}
h{font-weight:600;color:#323270;font-size:22px}
boxtag{font-size:13px;font-weight:600;border-style:solid;color:#fff;background-color:blue;border-color:blue;border-width:1px;margin-top:2px;padding-left:2px;padding-right:2px;border-radius:10px}
boxtag[type="awl"]{font-size:9px;font-weight:600;color:#fff;border-style:solid;border-width:1px;background-color:#000;border-color:#000;padding-left:1px;padding-right:1px;border-top:0;border-bottom:0}
vp-gs{display:none}
pron-g-blk{display:inline}
top-g{display:block}
pron-g-blk brelabel{padding-left:4px;font-size:14px}
pron-g-blk namelabel{padding-left:4px;font-size:14px}
pos xhtml\:a{display:table;color:#fff;font-weight:600;padding-left:2px;padding-right:2px;border-style:solid;border-width:1px;border-radius:5px;border-top:0;border-bottom:0;border-color:#c70000;background-color:#c70000}
vpform{color:#9b9b9b;font-style:italic}
vp-g{display:block;padding-left:12px}
sn-blk{display:block}
:not(idm-g) sn-gs sn-blk::before{padding-right:4px;counter-increment:sn_blk_counter;content:counter(sn_blk_counter)}
:not(id-g) sn-gs sn-blk::before{padding-right:4px;counter-increment:sn_blk_counter;content:counter(sn_blk_counter)}
def{font-weight:600}
xsymb{display:none}
xhtml\:br{}
x-g-blk{display:block;border-left:3px solid #dbdbdb;margin-left:8px;padding-left:10px}
x-g-blk x::before{content:'•'}
x-g-blk x{font-style:italic;color:#3784dd}
x-g-blk x chn{padding-left:13px;font-style:normal;color:#8d8d8d}
top-g xhtml\:br{display:none}
cf-blk{font-style:italic;font-weight:600;color:#2b7dca;padding-right:4px}
xr-gs{display:block}
xr-g-blk a:link{text-decoration:none;color:#a52a2a;font-weight:600}
def+x-gs cf-blk{font-style:italic;font-weight:600;color:#2b7dca;display:block}
shcut-blk{margin-top:14px;display:block;border-bottom:1px solid #a0a0a0;padding-bottom:5px}
gram-g{font-weight:600;color:#04b92b}
unbox{margin-top:16px;margin-bottom:16px;display:block;padding-left:5px;padding-right:15px;padding-top:10px;border:1px solid red;border-radius:12px}
unbox title{display:none}
unbox inlinelist{display:inline}
unbox inlinelist und{font-weight:600;color:#03648a}
unbox unsyn{display:block;font-weight:600;color:#1a4781}
unbox x-g-blk{display:block}
unbox x-g-blk x::before{content:'•';padding-right:6px}
unbox h3{color:#36866a;margin-bottom:4px;margin-top:6px}
unbox eb{font-weight:600}
pron-g-blk a:link{text-decoration:none}
audio-gbs-liju,audio-gb-liju,audio-brs-liju,audio-gb{padding-right:4px;color:blue;opacity:.8;display:none}
audio-uss-liju,audio-ams-liju,audio-us-liju,audio-us{padding-right:4px;color:#af0404;opacity:.8;display:none}
a:link{text-decoration:none}
eb{font-weight:600}
idm-gs un{display:block;color:#7c7070}
idm-blk idm{padding-top:12px;display:block;font-weight:600;color:#010102}
idm-g def{font-weight:500}
idm-g sn-blk::before{color:#6f49c7;content:'★'}
pv-g def{font-weight:500}
label-g-blk{color:#797979;font-style:italic}
pv-blk pv{padding-top:12px;display:block;font-weight:600;color:#1881e4}
unbox ul li{list-style-type:square}
unbox x-gs{display:block;margin-left:8px;padding-left:10px}
unbox x-gs chn{}
img{display:block;max-width:100%}
.big_pic{display:none;max-width:100%}
.switch_ec{display:none}
if-gs-blk{display:inline}
if-gs-blk form{display:inline}
unbox[type=wordfinder] xr-gs{display:inline}
unbox[type=wordfinder]::before{border:1px solid;border-radius:6px;position:relative;top:-24px;font-size:18px;color:#af1919;font-weight:600;content:'WordFinder';background-color:#fff;padding:5px 7px}
unbox[type=colloc]::before{border:1px solid;border-radius:6px;position:relative;top:-24px;left:18px;font-size:18px;color:#af1919;font-weight:600;content:'Collocations 词语搭配';background-color:#fff;padding:5px 7px}
unbox[type=wordfamily]{display:block;float:right}
unbox[type=wordfamily] wfw-g{display:block}
unbox[type=wordfamily] wfw-g wfw-blk{color:#101095;font-weight:600}
unbox[type=wordfamily] wfw-g wfo{font-weight:600}
unbox[type=wordfamily] wfw-g wfp-blk wfp{font-style:italic;color:#971717;font-weight:500}
unbox[type=wordfamily]::before{border:1px solid;border-radius:6px;position:relative;top:-24px;left:18px;font-size:18px;color:#af1919;font-weight:600;content:'WORD FAMILY';background-color:#fff;padding:5px 7px}
unbox[type=grammar]::before{border:1px solid;border-radius:6px;position:relative;top:-24px;left:18px;font-size:18px;color:#af1919;font-weight:600;content:'GRAMMAR 语法';background-color:#fff;padding:5px 7px}
unbox[type=grammar]{margin-top:36px}
unbox[type=grammar] x-gs{padding-left:0;margin-left:0}
unbox ul{margin-top:4px}
use-blk{color:#0b8a0b}
dis-g xr-gs{display:inline}
xr-gs[firstinblock="n"]{display:inline}
`

func ParseAddr(listen string) (network string, address string) {
	// Allow passing just -remote=auto, as a shorthand for using automatic remote
	// resolution.
	if listen == "auto" {
		return "auto", ""
	}
	if parts := strings.SplitN(listen, ";", 2); len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "tcp", listen
}
