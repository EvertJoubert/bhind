package bhservlet

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/EvertJoubert/bhind/bhdb"
	"github.com/EvertJoubert/bhind/bhio"
	"github.com/EvertJoubert/bhind/bhparameters"
	"github.com/EvertJoubert/bhind/bhscript"
)

type RequestSession interface {
}

//BhSession BhSession
type BhSession struct {
	*bhscript.BhScript
}

func (bhses *BhSession) Query(alias string, query string, args ...interface{}) *bhdb.DBQuery {
	return bhdb.DatabaseManager().Query(alias, query, args...)
}

func (bhses *BhSession) Execute(alias string, query string, args ...interface{}) *bhdb.DBExecuted {
	return bhdb.DatabaseManager().Execute(alias, query, args...)
}

func (bhses *BhSession) cleanupBhSession() {
	if bhses.BhScript != nil {
		bhses.BhScript.CleanupBhScript()
		bhses.BhScript = nil
	}
}

func newBHSession() *BhSession {
	return &BhSession{BhScript: bhscript.NewBhSrcipt()}
}

//BhServletRequest BhServletRequest
type BhServletRequest struct {
	*BhSession
	root                 string
	httpw                http.ResponseWriter
	httpr                *http.Request
	altw                 io.Writer
	altr                 io.Reader
	isaltrw              bool
	done                 chan bool
	session              *BhSession
	params               *bhparameters.Parameters
	nextRequests         []string
	nextRequesti         int
	nextResourceHandlers []BhResourceHandler
}

func (srvlet *BhServletRequest) parseParameters() {
	srvlet.params = &bhparameters.Parameters{}
	if err := srvlet.httpr.ParseMultipartForm(0); err == nil {
		if srvlet.httpr.MultipartForm != nil {
			for pname, pvalue := range srvlet.httpr.MultipartForm.Value {
				srvlet.params.SetParameter(pname, false, pvalue...)
			}
			for pname, pfile := range srvlet.httpr.MultipartForm.File {
				if len(pfile) > 0 {
					pfilei := []interface{}{}
					for pf, _ := range pfile {
						pfilei = append(pfilei, pf)
					}
					srvlet.params.SetFileParameter(pname, false, pfilei...)
					pfilei = nil
				}
			}
		}
	}
	if err := srvlet.httpr.ParseForm(); err == nil {
		if srvlet.httpr.PostForm != nil {
			for pname, pvalue := range srvlet.httpr.PostForm {
				srvlet.params.SetParameter(pname, false, pvalue...)
			}
		}
		if srvlet.httpr.Form != nil {
			for pname, pvalue := range srvlet.httpr.Form {
				srvlet.params.SetParameter(pname, false, pvalue...)
			}
		}
	}
}

//Parameters -> container of parameters of the current servlet being executed
func (srvlet *BhServletRequest) Parameters() *bhparameters.Parameters {
	return srvlet.params
}

//ContainsParameter -> check if parameter exist with given name
func (srvlet *BhServletRequest) ContainsParameter(pname string) bool {
	return srvlet.params != nil && srvlet.params.ContainsParameter(pname)
}

//Parameter -> return parameter value ([]string) with given name or nil
func (srvlet *BhServletRequest) Parameter(pname string, defaultval ...string) []string {
	if srvlet.params == nil {
		return defaultval
	}
	if srvlet.params.ContainsParameter(pname) {
		return srvlet.params.Parameter(pname)
	}
	return defaultval
}

//FileParameter -> return file parameter value ([]interface) with given name or nil
func (srvlet *BhServletRequest) FileParameter(pname string) []interface{} {
	if srvlet.params == nil {
		return nil
	}
	return srvlet.params.FileParameter(pname)
}

//ContainsFileParameter -> check if file parameter exist with given name
func (srvlet *BhServletRequest) ContainsFileParameter(pname string) bool {
	return srvlet.params != nil && srvlet.params.ContainsFileParameter(pname)
}

var markupExts = map[string]bool{".html": true, ".htm": true, ".xml": true, ".svg": true, ".js": false, ".json": false, ".txt": true, ".go": false}

func (bhSrvlt *BhServletRequest) AppendNextRequest(nextrequest ...string) {
	if bhSrvlt.nextRequests == nil {
		bhSrvlt.nextRequests = []string{}
	}
	var nextRsFormated func(string) string = func(nextrsToFormat string) string {

		return nextrsToFormat
	}
	if len(nextrequest) > 0 {
		for _, nextrs := range nextrequest {
			if strings.Index(nextrs, "|") > -1 {
				for _, nxtrs := range strings.Split(nextrs, "|") {
					bhSrvlt.nextRequests = append(bhSrvlt.nextRequests, nextRsFormated(nxtrs))
				}
			} else {
				bhSrvlt.nextRequests = append(bhSrvlt.nextRequests, nextRsFormated(nextrs))
			}
		}
	}
}

const timeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func (bhSrvlt *BhServletRequest) executeServlet() {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	if bhSrvlt.isaltrw {

	} else {
		path := bhSrvlt.httpr.URL.Path[1:]
		bhSrvlt.AppendNextRequest(path)
		istext, _ := isMarkupExt(path)
		bhSrvlt.parseParameters()
		bhSrvlt.AppendNextRequest(bhSrvlt.Parameters().Parameter("next-request")...)
		setmimetype := false
		seekFrom := int64(-1)
		rangevalue := bhSrvlt.httpr.Header.Get("Range")
		if rangevalue != "" {
			rangevalue = rangevalue[strings.Index(rangevalue, " ")+1:]
			if strings.HasPrefix(rangevalue, "bytes=") {
				rangevalue = rangevalue[len("bytes="):]
				rangevalue = strings.Split(rangevalue, "-")[0]
			}
			seekFrom, _ = strconv.ParseInt(rangevalue, 10, 64)
		}
		for len(bhSrvlt.nextRequests) > 0 {
			nextrequest := bhSrvlt.nextRequests[0]
			if strings.HasPrefix(nextrequest, "/") {
				nextrequest = nextrequest[1:]
			}
			if len(bhSrvlt.nextRequests) > 1 {
				bhSrvlt.nextRequests = bhSrvlt.nextRequests[1:]
			} else {
				bhSrvlt.nextRequests = nil
			}
			ext := filepath.Ext(nextrequest)
			if (ext == "" && len(bhSrvlt.nextRequests) == 0) || ext != "" {
				if resHndler := nextResourceHandler(bhSrvlt.root, nextrequest, bhSrvlt); resHndler != nil {
					defer resHndler.Close()
					if !setmimetype {

						if ext != "" || len(bhSrvlt.nextRequests) == 0 {
							defaultext := ".txt"

							mimetype, _ := FindMimeTypeByExt(ext, defaultext)

							if ext == ".mkv" {
								mimetype = "video/x-matroska"
							}

							if seekFrom == -1 {
								if strings.HasPrefix(mimetype, "video/") {
									seekFrom = 0
								}
							}
							if mimetype == "text/csv" {
								mimetype = "text/plain"
							}
							bhSrvlt.httpw.Header().Set("CONTENT-TYPE", mimetype)
							bhSrvlt.httpw.Header().Set("Last-Modified", time.Time{}.UTC().Format(timeFormat))
							if istext {
								bhSrvlt.httpw.Header().Set("Expires", time.Time{}.UTC().Format(timeFormat))
							}
							if istext || resHndler.handlerType == BhWidgetHandler || seekFrom == -1 {
								bhSrvlt.httpw.WriteHeader(200)
							}
							path = nextrequest
							setmimetype = true
						}
					}
					QueueResourceHandler(resHndler)
					//resHndler.prepHandler(bhSrvlt)
					if resHndler.handlerType != BhWidgetHandler && resHndler.pgrm == nil {
						reshndls := resHndler.Size()
						if seekFrom == -1 {
							resHndler.Seek(0, 0)
						} else {
							if !istext {
								bhSrvlt.httpw.Header().Set("Accept-Ranges", "bytes")
								bhSrvlt.httpw.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", seekFrom, reshndls-1, reshndls))
								bhSrvlt.httpw.WriteHeader(206)
							}
							resHndler.Seek(seekFrom, 0)
							seekFrom = -1
						}

						io.CopyN(bhSrvlt.httpw, resHndler, reshndls)
					}
				}
			}
		}
	}
}

func (bhSrvlt *BhServletRequest) CleanupBhServletRequest() {
	if bhSrvlt.altr != nil {
		bhSrvlt.altr = nil
	}
	if bhSrvlt.altw != nil {
		bhSrvlt.altw = nil
	}
	if bhSrvlt.done != nil {
		bhSrvlt.done = nil
	}
	if bhSrvlt.httpr != nil {
		bhSrvlt.httpr = nil
	}
	if bhSrvlt.httpw != nil {
		bhSrvlt.httpw = nil
	}
	if bhSrvlt.BhSession != nil {
		bhSrvlt.BhSession.cleanupBhSession()
		bhSrvlt.BhSession = nil
	}
	if bhSrvlt.params != nil {
		bhSrvlt.params.CleanupParameters()
		bhSrvlt.params = nil
	}
}

func isMarkupExt(ext string) (isext bool, isjsext bool) {
	ext = filepath.Ext(ext)
	isjsext, isext = markupExts[ext]
	if isext {
		if isjsext {
			isjsext = false
		} else {
			isjsext = true
		}
	} else {
		isjsext = false
	}
	return isext, isjsext
}

func DummyHttpRequestResponse(wr io.Writer, writeheaders bool, protocol string, uri string, method string, requestContent *bhio.BhIORW, headersparams ...string) (w http.ResponseWriter, r *http.Request, err error) {
	if protocol == "" {
		protocol = "HTTP/1.1"
	}
	if method == "" {

		method = "GET"
	}
	if uri == "" {
		uri = "/"
	}
	r = new(http.Request)

	r.Method = method
	r.Proto = protocol
	var ok bool
	r.ProtoMajor, r.ProtoMinor, ok = http.ParseHTTPVersion(r.Proto)
	if !ok {
		if r != nil {
			r = nil
		}
		return w, r, errors.New("servlet: invalid PROTOCOL version")
	}

	r.Close = true
	r.Trailer = http.Header{}
	r.Header = http.Header{}

	r.Host = "localhost"

	qparams := ""

	if strings.Index(uri, "?") > strings.LastIndex(uri, "/") {
		qparams = uri[strings.Index(uri, "?"):]
		uri = uri[:strings.Index(uri, "?")]
		if qparams != "" {
			qparams = url.QueryEscape(qparams)
		}
	}

	if len(headersparams) > 0 {
		for _, header := range headersparams {
			if strings.Index(header, "=") > 0 {
				if strings.HasPrefix(header, "header:") {
					hname := strings.TrimSpace(header[len("header:"):])
					hval := strings.TrimSpace(hname[strings.Index(hname, "=")+1:])
					hname = strings.TrimSpace(hname[:strings.Index(hname, "=")])
					if hname != "" && hval != "" {
						r.Header.Add(hname, hval)
					}
				}
			}
		}
	}

	if qparams != "" {
		rawurl := "http://" + r.Host + uri + "?" + qparams
		urlparsed, urlparsederr := url.Parse(rawurl)
		if urlparsederr != nil {
			if r != nil {
				r = nil
			}
			return w, r, urlparsederr
		}
		r.URL = urlparsed
	} else {
		rawurl := "http://" + r.Host + uri
		urlparsed, urlparsederr := url.Parse(rawurl)
		if urlparsederr != nil {
			if r != nil {
				r = nil
			}
			return w, r, urlparsederr
		}
		r.URL = urlparsed
	}

	internalrcontent := false

	if requestContent == nil {
		boundary := ""
		if len(headersparams) > 0 {
			for _, parameter := range headersparams {
				if strings.Index(parameter, "=") > 0 {
					if strings.HasPrefix(parameter, "param:") {
						if !internalrcontent {
							internalrcontent = true
							requestContent, _ = bhio.NewBhIORW()
							boundary = randomBoundary()

						}
						pname := strings.TrimSpace(parameter[len("param:"):])
						pval := strings.TrimSpace(pname[strings.Index(pname, "=")+1:])
						pname = strings.TrimSpace(pname[:strings.Index(pname, "=")])
						if pname != "" {
							requestContent.Println()
							requestContent.Println("--" + boundary)
							requestContent.Println("Content-Disposition: form-data; name=\"" + pname + "\"")
							requestContent.Println()
							requestContent.Print(pval)
						}
					}
				}
			}
			if internalrcontent {
				requestContent.Println()
				requestContent.Print("--" + boundary + "--")
				r.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)
			}
		}
	}

	if requestContent != nil {
		if requestContent.Size() > 0 {
			if r.Method != "POST" {
				r.Method = "POST"
			}
			if r.Header.Get("Content-Length") == "" {
				r.Header.Set("Content-Length", fmt.Sprintf("%d", requestContent.Size()))
				r.ContentLength = requestContent.Size()
			}

			if r.ContentLength > 0 {
				r.Body = ioutil.NopCloser(io.LimitReader(requestContent, r.ContentLength))
			}
		}
		requestContent = nil
	}

	return &response{
		req:            r,
		header:         make(http.Header),
		bufw:           bufio.NewWriter(wr),
		canwriteheader: writeheaders,
		requestContent: requestContent,
	}, r, err
}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

type response struct {
	req            *http.Request
	header         http.Header
	bufw           *bufio.Writer
	headerSent     bool
	canwriteheader bool
	requestContent *bhio.BhIORW
}

func (r *response) Flush() {
	r.bufw.Flush()
}

func (r *response) Header() http.Header {
	return r.header
}

func (r *response) Write(p []byte) (n int, err error) {
	if !r.headerSent {
		r.WriteHeader(http.StatusOK)
	}
	return r.bufw.Write(p)
}

func (r *response) WriteHeader(code int) {
	if r.headerSent {
		// Note: explicitly using Stderr, as Stdout is our HTTP output.
		fmt.Fprintf(os.Stderr, "servlet attempted to write header twice on request for %s", r.req.URL)
		return
	}
	r.headerSent = true
	if r.canwriteheader {
		fmt.Fprintf(r.bufw, "Status: %d %s\r\n", code, http.StatusText(code))

		// Set a default Content-Type
		if _, hasType := r.header["Content-Type"]; !hasType {
			r.header.Add("Content-Type", "text/html; charset=utf-8")
		}

		r.header.Write(r.bufw)
		r.bufw.WriteString("\r\n")
		r.bufw.Flush()
	}
}

func (bhSrvlt *BhServletRequest) executeHTTPRW(session *BhSession) {
	bhSrvlt.altw = bhSrvlt.httpw
	bhSrvlt.executeIoRW(session)
}

func (bhSrvlt *BhServletRequest) executeIoRW(session *BhSession) {

}

//DefaultHTTPHandle DefaultHTTPHandle
func DefaultHTTPHandle(root string, w http.ResponseWriter, r *http.Request) (err error) {
	defer runtime.GC()
	rw, rwok := w.(*response)
	queueHTTPRW(rwok, root, w, r)
	if rwok {
		defer func() {
			if rw.requestContent != nil {
				rw.requestContent.Close()
				rw.requestContent = nil
			}
		}()
		rw.Write(nil) // make sure a response is sent
		err = rw.bufw.Flush()

	}
	return err
}

//Listen Listen
func Listen(root string, addresses ...string) {
	for _, addr := range addresses {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			DefaultHTTPHandle(root, w, r)
		})
		server := &http.Server{
			Addr:         addr,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  30 * time.Second,
			Handler:      mux,
		}
		go func() {
			server.ListenAndServe()
		}()
	}
}

var queuedServlet chan *BhServletRequest
var queuedServletsLock *sync.Mutex = &sync.Mutex{}

func init() {
	if queuedServlet == nil {
		queuedServlet = make(chan *BhServletRequest)
		go func() {
			for {
				select {
				case bhSvlt := <-queuedServlet:
					go func() {
						defer func() {
							bhSvlt.done <- true
						}()
						bhSvlt.executeServlet()
					}()
				default:
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}
}

func queueServlets(bhSvlt *BhServletRequest) {
	queuedServlet <- bhSvlt

	if <-bhSvlt.done {
		close(bhSvlt.done)
		bhSvlt.CleanupBhServletRequest()
	}
}

func queueHTTPRW(localRequest bool, root string, w http.ResponseWriter, r *http.Request) {
	root = strings.Replace(root, "\\", "/", -1)
	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}
	bhsvrlt := &BhServletRequest{root: root, BhSession: newBHSession(), params: bhparameters.NewParameters(), isaltrw: false, httpw: w, httpr: r, done: make(chan bool, 1), nextRequesti: -1}
	bhsvrlt.BhSession.BhScript.Bound("_servlet", bhsvrlt)
	if localRequest {
		bhsvrlt.BhSession.BhScript.Bound("_dbmanager", bhdb.DatabaseManager())
		bhsvrlt.BhSession.BhScript.Bound("Listen", Listen)
	}
	queueServlets(bhsvrlt)
}

func queueIoRW(w io.Writer, r io.Reader) {
	bhsvrlt := &BhServletRequest{BhSession: newBHSession(), params: bhparameters.NewParameters(), isaltrw: true, altw: w, altr: r, done: make(chan bool, 1), nextRequesti: -1}
	bhsvrlt.BhSession.BhScript.Bound("_parameters", bhsvrlt.params)
	queueServlets(bhsvrlt)
}

func LoadSystemConfig(rootPath, sname string, checkEveryXSeconds time.Duration, wr io.Writer, params ...string) {
	go func() {
		lastMod := time.Now()
		for {
			select {
			default:
				if res := RegisteredResource(rootPath, sname+".js"); res != nil {
					if res.LastModified() != lastMod {
						if w, r, err := DummyHttpRequestResponse(wr, false, "", "/"+sname+".js", "", nil, params...); err == nil {
							DefaultHTTPHandle(rootPath, w, r)
							lastMod = res.LastModified()
						} else {
							fmt.Println(err.Error())
						}
					}
				} else if !RegisteredResourceExists(rootPath, sname+".js") {
					if w, r, err := DummyHttpRequestResponse(wr, false, "", "/"+sname+".js", "", nil, params...); err == nil {
						DefaultHTTPHandle(rootPath, w, r)
						if res = RegisteredResource(rootPath, sname+".js"); res != nil {
							lastMod = res.LastModified()
						}
					} else {
						fmt.Println(err.Error())
					}
				}
				if checkEveryXSeconds < 5 {
					checkEveryXSeconds = 5
				}
				time.Sleep(checkEveryXSeconds * time.Second)
			}
		}
	}()
}
