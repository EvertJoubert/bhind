package bhservlet

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"

	"github.com/EvertJoubert/bhind/bhio"
)

type BhHandlerType int

const BhFileHandler BhHandlerType = 0
const BhEmbeddedHandler BhHandlerType = 1
const BhWidgetHandler BhHandlerType = 2

type BhResourceHandler struct {
	*BhWidget
	bhIORW       *bhio.BhIORW
	bhiocur      *bhio.BhReadWriteCursor
	handlerType  BhHandlerType
	reshndlTime  time.Time
	invokeWidget func(path string, srvlt *BhServletRequest) WidgetInterface
	activeWidget WidgetInterface
	done         chan bool
	//srvlt        *BhServletRequest
	//path         string
	out  *bhio.BhIORW
	pgrm *goja.Program
}

func (bhrshndlr *BhResourceHandler) Print(a ...interface{}) {
	if bhrshndlr.out != nil {
		bhrshndlr.out.Print(a...)
	}
}

func (bhrshndlr *BhResourceHandler) Println(a ...interface{}) {
	if bhrshndlr.out != nil {
		bhrshndlr.out.Println(a...)
	}
}

func (bhrshndlr *BhResourceHandler) Read(p []byte) (n int, err error) {
	if bhrshndlr.activeWidget != nil {
		n, err = bhrshndlr.activeWidget.Read(p)
	} else if bhrshndlr.bhiocur != nil {
		n, err = bhrshndlr.bhiocur.Read(p)
	} else {
		err = io.EOF
	}
	return n, err
}

func (bhrshandlr *BhResourceHandler) OutputContentByPos(startpos int64, endpos int64) {
	if bhrshandlr.bhiocur != nil {
		bhrshandlr.bhiocur.Seek(startpos, 0)
		totalreadlen := endpos - startpos

		rbl := 4096
		if totalreadlen < 4096 {
			rbl = int(totalreadlen)
		}
		rb := make([]byte, rbl)

		for totalreadlen > 0 {
			rbl := 4096
			if totalreadlen < 4096 {
				rbl = int(totalreadlen)
			}
			if len(rb) != rbl {
				rb = nil
				rb = make([]byte, rbl)
			}
			rn, rnerr := bhrshandlr.bhiocur.Read(rb[0:rbl])
			if rn > 0 {
				rni := 0
				for rni < rn {
					if wn, wnerr := bhrshandlr.Write(rb[rni:rn]); wnerr == nil {
						if wn > 0 {
							rni += wn
						}
					} else if wnerr != nil {
						if rnerr == nil || rnerr == io.EOF {
							rnerr = wnerr
						}
						break
					}
				}
				totalreadlen -= int64(rn)
			} else {
				if rnerr == nil {
					rnerr = io.EOF
				} else if rnerr == io.EOF {
					break
				}
			}
			if rnerr != nil && rnerr != io.EOF {
				break
			}
		}
	}
}

func (bhrshndlr *BhResourceHandler) Write(p []byte) (n int, err error) {
	if bhrshndlr.srvlt.httpw != nil {
		n, err = bhrshndlr.srvlt.httpw.Write(p)
	} else if bhrshndlr.srvlt.altw != nil {
		n, err = bhrshndlr.srvlt.altw.Write(p)
	}
	return n, err
}

func (bhrshndlr *BhResourceHandler) Seek(offset int64, whence int) (n int64, err error) {
	if bhrshndlr.activeWidget != nil {
		n, err = bhrshndlr.activeWidget.Seek(offset, whence)
	} else if bhrshndlr.bhiocur != nil {
		n, err = bhrshndlr.bhiocur.Seek(offset, whence)
	}
	return n, err
}

func (bhrshndlr *BhResourceHandler) Size() (n int64) {
	n, _ = bhrshndlr.Seek(0, 2)
	return n
}

func (bhrshndlr *BhResourceHandler) Close() (err error) {
	if bhrshndlr.bhiocur != nil {
		bhrshndlr.bhiocur.Close()
		bhrshndlr.bhiocur = nil
	}
	if bhrshndlr.bhIORW != nil {
		bhrshndlr.bhIORW.Close()
		bhrshndlr.bhIORW = nil
	}
	if bhrshndlr.activeWidget != nil {
		bhrshndlr.activeWidget.DisposeWidget()
		bhrshndlr.activeWidget = nil
	}
	if bhrshndlr.srvlt != nil {
		bhrshndlr.srvlt = nil
	}
	if bhrshndlr.done != nil {
		close(bhrshndlr.done)
		bhrshndlr.done = nil
	}
	if bhrshndlr.out != nil {
		bhrshndlr.out = nil
	}
	if bhrshndlr.pgrm != nil {
		bhrshndlr.pgrm = nil
	}
	/*if bhrshndlr.srvlt != nil {
		bhrshndlr.srvlt = nil
	}*/
	if bhrshndlr.BhWidget != nil {
		bhrshndlr.BhWidget.DisposeWidget()
		bhrshndlr.BhWidget = nil
	}
	return err
}

func (bhrshnldr *BhResourceHandler) prepHandler(bhsrvlt *BhServletRequest) {
	if bhrshnldr.invokeWidget != nil {
		bhrshnldr.activeWidget = bhrshnldr.invokeWidget(bhrshnldr.path, bhsrvlt)
		bhrshnldr.activeWidget.InvokeWidget()
		bhrshnldr.activeWidget.LoadWidget()
		if bhrshnldr.pgrm != nil {
			bhsrvlt.Bound("_hndlr", bhrshnldr)
			bhsrvlt.Bound("_widget", bhrshnldr.activeWidget)
			if err := bhsrvlt.Eval(bhrshnldr.pgrm); err != nil {
				fmt.Println(err)
			}
		} else {
			bhrshnldr.activeWidget.ProcessWidget()
		}
		bhrshnldr.activeWidget.UnloadWidget()
	} else if bhrshnldr.pgrm != nil {
		//bhrshnldr.activeWidget = NewBhWidget(bhrshnldr.path, bhsrvlt)
		//bhrshnldr.activeWidget.InvokeWidget()
		//bhrshnldr.activeWidget.LoadWidget()
		bhsrvlt.Bound("_hndlr", bhrshnldr)
		bhsrvlt.Bound("_widget", bhrshnldr)
		if err := bhsrvlt.Eval(bhrshnldr.pgrm); err != nil {
			fmt.Println(err)
		}
		//bhrshnldr.activeWidget.UnloadWidget()
		//bhrshnldr.activeWidget.DisposeWidget()
		//bhrshnldr.activeWidget = nil
	}
}

func nextResourceHandler(root string, path string, srvlt *BhServletRequest) (bhrshndlr *BhResourceHandler) {
	bhrshndlr = &BhResourceHandler{BhWidget: NewBhWidget(path, srvlt), reshndlTime: time.Now(), done: make(chan bool, 1)}
	bhrshndlr.BhWidget.InvokeWidget()
	if strings.HasPrefix(root, "http://") || strings.HasPrefix(root, "https://") {

	} else {
		if res := nextResource(path); res != nil {
			bhrshndlr.reshndlTime = res.modified
			if res.invokeWidget != nil {
				bhrshndlr.invokeWidget = res.invokeWidget
				bhrshndlr.handlerType = BhWidgetHandler
			} else {
				if res.resscrpt == nil {
					bhrshndlr.bhiocur = res.ioRW.ReadWriteCursor(true)
					if bhrshndlr.bhiocur != nil && bhrshndlr.bhiocur.FileInfo() != nil {
						bhrshndlr.handlerType = BhFileHandler
					} else {
						bhrshndlr.handlerType = BhEmbeddedHandler
					}
				} else {
					if res.resscrpt.hasContent() {
						bhrshndlr.bhiocur = res.resscrpt.content().ReadWriteCursor(true)
					}
					if res.resscrpt.hasCode() {
						if res.resscrpt.hasContent() {
							if bhrshndlr.out != nil {
								bhrshndlr.out = nil
							}
							if bhrshndlr.srvlt.httpw != nil {
								bhrshndlr.out, _ = bhio.NewBhIORW(bhrshndlr.srvlt.httpw)
							} else if bhrshndlr.srvlt.altw != nil {
								bhrshndlr.out, _ = bhio.NewBhIORW(bhrshndlr.srvlt.altw)
							}
							if bhrshndlr.out != nil {
								bhrshndlr.srvlt.Bound("_out", bhrshndlr.out)
							}
						}
						bhrshndlr.pgrm = res.resscrpt.prg
					}
				}
			}
		} else {
			if finfo, finfoerr := os.Stat(root + path); finfoerr == nil {
				if finfo.IsDir() {
					var files []string
					bhrshndlr.bhIORW, _ = bhio.NewBhIORW(nil, nil)
					root = root + path
					if !strings.HasSuffix(root, "/") {
						root = root + "/"
					}
					if err := filepath.Walk(root, func(fpath string, info os.FileInfo, err error) error {
						file := strings.Replace(fpath, "\\", "/", -1)
						if strings.HasPrefix(file, root) && file != root && strings.LastIndex(file[len(root):], "/") == -1 {
							files = append(files, file[len(root):])
						}
						return nil
					}); err == nil {
						for _, fname := range files {
							if path != "" {
								if !strings.HasSuffix(path, "/") {
									path = path + "/"
								}
							}
							bhrshndlr.bhIORW.Println("<a href=\"/" + path + fname + "\">" + fname + "</a><br/>")
						}
					}
				} else {
					//RegisterResourceByReader(root, path, finfo)
					if res := registerResourceFromRootPath(root, path); res != nil {
						bhrshndlr.reshndlTime = res.modified
						if res.invokeWidget != nil {
							bhrshndlr.invokeWidget = res.invokeWidget
							bhrshndlr.handlerType = BhWidgetHandler
						} else {
							if res.resscrpt == nil {
								bhrshndlr.bhiocur = res.ioRW.ReadWriteCursor(true)
								if bhrshndlr.bhiocur != nil && bhrshndlr.bhiocur.FileInfo() != nil {
									bhrshndlr.handlerType = BhFileHandler
								} else {
									bhrshndlr.handlerType = BhEmbeddedHandler
								}
							}
						}
						if res.resscrpt != nil {
							if res.resscrpt.hasContent() {
								bhrshndlr.bhiocur = res.resscrpt.content().ReadWriteCursor(true)
							}
							if res.resscrpt.hasCode() {
								if res.resscrpt.hasContent() {
									if bhrshndlr.out != nil {
										bhrshndlr.out = nil
									}
									if bhrshndlr.srvlt.httpw != nil {
										bhrshndlr.out, _ = bhio.NewBhIORW(bhrshndlr.srvlt.httpw)
									} else if bhrshndlr.srvlt.altw != nil {
										bhrshndlr.out, _ = bhio.NewBhIORW(bhrshndlr.srvlt.altw)
									}
									if bhrshndlr.out != nil {
										bhrshndlr.srvlt.Bound("_out", bhrshndlr.out)
									}
								}
								bhrshndlr.pgrm = res.resscrpt.prg
							}
						}
					}
				}
			}
		}
	}
	if bhrshndlr != nil {
		if bhrshndlr.bhIORW != nil {
			if bhrshndlr.bhiocur == nil {
				bhrshndlr.bhiocur = bhrshndlr.bhIORW.ReadWriteCursor(true)
			}
		}
	}
	return bhrshndlr
}

func registerResourceFromRootPath(root string, path string) (res *BhResource) {
	if finfo, finfoerr := os.Stat(root + path); finfoerr == nil && !finfo.IsDir() {
		RegisterResourceByReader(root, path, finfo)
		res = nextResource(path)
	}
	return res
}

var resourceWidgetInvokes map[string]func(string, *BhServletRequest) WidgetInterface = make(map[string]func(string, *BhServletRequest) WidgetInterface)
var resourceWidgetInvokesLock *sync.Mutex = &sync.Mutex{}

var resources map[string]*BhResource = make(map[string]*BhResource)
var resourceLock *sync.RWMutex = &sync.RWMutex{}

func MonitorResources() {
	monRes := make(chan func(), 1)
	var monResources func() = func() {
		if len(resources) > 0 {
			for respath, res := range resources {
				if !res.isValid() {
					resourceLock.RLock()
					deleteResource(res)
					res = nil
					delete(resources, respath)
					resourceLock.RUnlock()
				}
			}
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
	}

	go func() {
		monRes <- monResources
		for {
			select {
			case mres := <-monRes:
				go func() {
					mres()
					monRes <- monResources
				}()
			}
		}
	}()
}

type BhResource struct {
	ioRW            *bhio.BhIORW
	processedActive bool
	isext           bool
	isjsext         bool
	path            string
	root            string
	resLock         *sync.RWMutex
	modified        time.Time
	resModel        *BhResourceModel
	invokeWidget    func(string, *BhServletRequest) WidgetInterface
	resscrpt        *BhResourceScript
}

func (bhres *BhResource) cleanupBhResource() {
	if bhres.ioRW != nil {
		bhres.ioRW.Close()
		bhres.ioRW = nil
	}
	if bhres.resModel != nil {
		bhres.resModel = nil
	}
	if bhres.resscrpt != nil {
		bhres.resscrpt.cleanupBhResourceScript()
		bhres.resscrpt = nil
	}
	if bhres.invokeWidget != nil {
		bhres.invokeWidget = nil
	}
	if bhres.resLock != nil {
		bhres.resLock = nil
	}
}

func (bhres *BhResource) LastModified() time.Time {
	return bhres.modified
}

func (bhres *BhResource) isValid() (valid bool) {
	valid = false
	bhres.resLock.RLock()
	defer bhres.resLock.RUnlock()
	if bhres.invokeWidget != nil {
		valid = true
	} else if finfo := bhres.ioRW.FileInfo(); finfo != nil {
		if cfinfo, cfinfoerr := os.Stat(bhres.root + bhres.path); cfinfoerr == nil && cfinfo.ModTime() == finfo.ModTime() {
			valid = true
		} else {
			if cfinfoerr != nil {
				valid = false
			} else if cfinfo.ModTime() != finfo.ModTime() {
				bhres.resLock.RLock()
				defer bhres.resLock.RUnlock()

				if bhres.ioRW != nil {
					bhres.ioRW.Close()
					bhres.ioRW = nil
				}
				if iorw, iorwerr := bhio.NewBhIORW(cfinfo, bhres.root+bhres.path); iorwerr == nil {
					bhres.ioRW = iorw
					if bhres.isext && bhres.resscrpt != nil {
						bhres.resscrpt.cleanupBhResourceScript()
						bhres.resscrpt = nil
					}

					bhres.resscrpt = &BhResourceScript{res: bhres}
					if err := bhres.resscrpt.loadScript(bhres.isjsext); err == nil {
						if bhres.resscrpt.empty() || bhres.resscrpt.contentOnly() {
							bhres.resscrpt.cleanupBhResourceScript()
							bhres.resscrpt = nil
						}
					} else {
						bhres.resscrpt.cleanupBhResourceScript()
						bhres.resscrpt = nil
					}

					bhres.modified = cfinfo.ModTime()
					valid = true
				} else {
					valid = false
				}
			}
		}
	} else {
		valid = true
	}

	return valid
}

func deleteResource(res *BhResource) {
	res.resLock.Lock()
	defer func() {
		res.resLock.Unlock()
		res.cleanupBhResource()
	}()
	resourceLock.RLock()
	defer resourceLock.RUnlock()
	delete(resources, res.path)
}

func nextResource(path string) (res *BhResource) {
	resourceLock.RLock()
	defer resourceLock.RUnlock()
	if _, resExists := resources[path]; resExists {
		res = resources[path]
		if !res.isValid() {
			deleteResource(res)
			res = nil
		}
	}
	return res
}

func newResource(root string, path string, r interface{}) (res *BhResource) {
	isext, isjsext := isMarkupExt(filepath.Ext(path))
	if widgetinv, widgetinvok := resourceWidgetInvokes[path]; widgetinvok {
		res = &BhResource{isext: isext, isjsext: isjsext, root: root, path: path, resLock: &sync.RWMutex{}, invokeWidget: widgetinv, modified: time.Now()}
	}
	if rr, rrok := r.(io.Reader); rrok {
		ioRW, _ := bhio.NewBhIORW()
		ioRW.Print(rr)
		res = &BhResource{isext: isext, isjsext: isjsext, root: root, path: path, resLock: &sync.RWMutex{}, ioRW: ioRW, modified: time.Now()}
	} else if finfo, finfook := r.(os.FileInfo); finfook {
		if iorw, iorwerr := bhio.NewBhIORW(finfo, root+path); iorwerr == nil {
			res = &BhResource{isext: isext, isjsext: isjsext, root: root, path: path, resLock: &sync.RWMutex{}, ioRW: iorw, modified: finfo.ModTime()}
		}
	}

	if res != nil {
		if isext {
			if res.ioRW != nil {
				res.resModel = newResourceModel(res)
				res.resscrpt = &BhResourceScript{res: res}
				if err := res.resscrpt.loadScript(res.isjsext); err == nil {
					if res.resscrpt.empty() || res.resscrpt.contentOnly() {
						res.resscrpt.cleanupBhResourceScript()
						res.resscrpt = nil
					}
				}
			}
		}
	}

	return res
}

func RegisterResourceByReader(root string, path string, r interface{}) {
	resourceLock.RLock()
	defer resourceLock.RUnlock()
	if _, resExists := resources[path]; !resExists {
		if res := newResource(root, path, r); res != nil {
			if res.isValid() {
				resources[path] = res
			} else {
				deleteResource(res)
			}
		}
	}
}

func RegisterResource(root string, a ...interface{}) {
	resourceLock.Lock()
	defer resourceLock.Unlock()
	if len(a) > 0 && len(a)%2 == 0 {
		for len(a) > 0 {
			if s, sok := a[0].(string); sok {
				if _, resExists := resources[s]; !resExists {
					if res := newResource(root, s, a[1]); res != nil {
						if res.isValid() {
							resources[s] = res
						} else {
							deleteResource(res)
						}
					}
				}
			}
			if len(a) > 2 {
				a = a[2:]
			} else {
				a = nil
			}
		}
	}
}

func RegisteredResource(root string, resname string) (res *BhResource) {
	if len(resources) > 0 {
		if _, exists := resources[resname]; exists {
			if res = resources[resname]; res.root != root {
				res = nil
			} else if !res.isValid() {
				deleteResource(res)
				res = nil
			}
		}
	}
	return res
}

func RegisteredResourceExists(root string, resname string) (exists bool) {
	resourceLock.RLock()
	defer resourceLock.RUnlock()
	if len(resources) > 0 {
		_, exists = resources[resname]
	}
	return exists
}

func RegisterWidgetInvoke(path string, invokeWidget func(string, *BhServletRequest) WidgetInterface) {
	resourceWidgetInvokesLock.Lock()
	defer resourceWidgetInvokesLock.Unlock()
	if _, resExists := resources[path]; !resExists {
		resourceWidgetInvokes[path] = invokeWidget
		if res := newResource("", path, nil); res != nil && res.isValid() {
			resources[path] = res
		}
	}
}

func RegisterWidget(path string, events ...interface{}) {
	resourceWidgetInvokesLock.Lock()
	defer resourceWidgetInvokesLock.Unlock()
	if _, resExists := resources[path]; !resExists {
		resourceWidgetInvokes[path] = func(path string, srvlet *BhServletRequest) WidgetInterface {
			return NewBhWidget(path, srvlet, events)
		}
		if res := newResource("", path, nil); res != nil && res.isValid() {
			resources[path] = res
		}
	}
}

const tagstartregexp string = `^((.:(([a-z]|[A-Z])\w*)+)|(([a-z]|[A-Z])+(:(([a-z]|[A-Z])\w*)+)+))+(:(([a-z]|[A-Z])\w*)+)*(-(([a-z]|[A-Z])\w*)+)?(.([a-z]|[A-Z])+)?$`

var regexptagstart *regexp.Regexp

const propregexp string = `^-?-?(([a-z]+[0-9]*)[a-z]*)+(-([a-z]+[0-9]*)[a-z]*)?$`

var regexprop *regexp.Regexp

const propvalnumberexp string = `^[-+]?\d+([.]\d+)?$`

var regexpropvalnumberexp *regexp.Regexp

var queuedResourceHandlers chan *BhResourceHandler
var queuedResourceHandlersLock *sync.Mutex = &sync.Mutex{}

func init() {

	if regexptagstart == nil {
		regexptagstart = regexp.MustCompile(tagstartregexp)
	}
	if regexprop == nil {
		regexprop = regexp.MustCompile(propregexp)
	}

	if regexpropvalnumberexp == nil {
		regexpropvalnumberexp = regexp.MustCompile(propvalnumberexp)
	}

	if queuedResourceHandlers == nil {
		queuedResourceHandlers = make(chan *BhResourceHandler)
		go func() {
			for {
				select {
				case reshndlr := <-queuedResourceHandlers:
					go func() {
						defer func() { reshndlr.done <- true }()
						reshndlr.prepHandler(reshndlr.srvlt)
					}()
				default:
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}
}

func QueueResourceHandler(reshndlr *BhResourceHandler) {
	queuedResourceHandlers <- reshndlr
	<-reshndlr.done
	close(reshndlr.done)
}

type BhResourceScript struct {
	prg                *goja.Program
	prgerr             error
	parseTokensChannel chan *bhResScriptToken
	res                *BhResource
	cntnt              *bhio.BhIORW
	cde                *bhio.BhIORW
}

func (bhresscrpt *BhResourceScript) content() *bhio.BhIORW {
	if bhresscrpt.cntnt == nil {
		bhresscrpt.cntnt, _ = bhio.NewBhIORW()
	}
	return bhresscrpt.cntnt
}

func flushcontent(bhresscrpt *BhResourceScript, token *bhResScriptToken, prevToken *bhResScriptToken, forcematchedio bool, r io.Reader, parked bool) {
	if r != nil {
		if parked {
			token.passiveAdhocIO().Print(r)
		} else {
			if prevToken == nil {
				if forcematchedio || token.hasAtvMatched {
					if !token.hasAtvMatched {
						token.hasAtvMatched = true
					}
					contentpos := bhresscrpt.content().Size()
					bhresscrpt.content().Print(r)
					nextcontentpos := bhresscrpt.content().Size()
					if contentpos > -1 && nextcontentpos > -1 && contentpos < nextcontentpos {
						bhresscrpt.code().Print(fmt.Sprintf("_hndlr.OutputContentByPos(%d,%d);", contentpos, nextcontentpos))
					}
				} else {
					bhresscrpt.content().Print(r)
				}
			} else {
				token.generalIO().Print(r)
			}
		}
	}
}

func (bhresscrpt *BhResourceScript) code() *bhio.BhIORW {
	if bhresscrpt.cde == nil {
		bhresscrpt.cde, _ = bhio.NewBhIORW()
	}
	return bhresscrpt.cde
}

func flushcode(bhresscrpt *BhResourceScript, token *bhResScriptToken, prevToken *bhResScriptToken, iorw *bhio.BhIORW, parked bool) {
	if iorw != nil {
		if !iorw.Empty() {
			if parked {
				if prevToken == nil {
					token.passiveAdhocIO().Print(token.atvlabels[0])
				} else {
					token.passiveAdhocIO().Print(prevToken.atvlabels[0])
				}
				token.passiveAdhocIO().Print(iorw)
				if prevToken == nil {
					token.passiveAdhocIO().Print(token.atvlabels[1])
				} else {
					token.passiveAdhocIO().Print(prevToken.atvlabels[1])
				}
			} else {
				if prevToken == nil {
					bhresscrpt.code().Print(iorw)
				} else {
					token.generalIO().Print(prevToken.atvlabels[0])
					token.generalIO().Print(iorw)
					token.generalIO().Print(prevToken.atvlabels[1])
				}
			}
			iorw.Close()
		}
	}
}

type bhResScriptToken struct {
	bhresscrpt     *BhResourceScript
	prevToken      *bhResScriptToken
	isjtext        bool
	atvlabels      []string
	atvlabelsi     []int
	prvAtvb        byte
	path           string
	pathname       string
	simplepathname string
	pathext        string
	atvPreIORWs    []*bhio.BhIORW
	atvUnmatchedIO *bhio.BhIORW
	//atvParkedUnmatchedIO *bhio.BhIORW
	atvMatchedIO       *bhio.BhIORW
	hasAtvMatched      bool
	genIO              *bhio.BhIORW
	atvParkedMatchedIO *bhio.BhIORW
	atvb               []byte
	psvlabels          []string
	psvlabelsi         []int
	prvPsvb            byte
	psvUnmatchedIO     *bhio.BhIORW
	psvFlushableIO     *bhio.BhIORW
	psvMatchedIO       *bhio.BhIORW
	psvValidMatchIO    *bhio.BhIORW
	psvParkedMatchedIO *bhio.BhIORW
	psvAdhocIO         *bhio.BhIORW
	psvParkedAdhocIO   *bhio.BhIORW
	psvb               []byte
	iocur              *bhio.BhReadWriteCursor
	tokenstage         int
	psvElemFound       string
	psvElemFoundCount  int
	psvPropFound       string
	psvProperties      map[string]string
}

func (ressrcptoken *bhResScriptToken) possibleProperties() map[string]string {
	if ressrcptoken.psvProperties == nil {
		ressrcptoken.psvProperties = make(map[string]string)
	}
	return ressrcptoken.psvProperties
}

//ACTIVE

func (resscrptoken *bhResScriptToken) activeIO() *bhio.BhIORW {
	if resscrptoken.atvPreIORWs == nil || len(resscrptoken.atvPreIORWs) == 0 {
		return resscrptoken.incActiveIO()
	}
	return resscrptoken.atvPreIORWs[len(resscrptoken.atvPreIORWs)-1]
}

func (resscrptoken *bhResScriptToken) printactiveIO() {
	if resscrptoken.atvPreIORWs != nil && len(resscrptoken.atvPreIORWs) > 0 {
		for _, iorw := range resscrptoken.atvPreIORWs {
			fmt.Println(iorw)
		}
	}
}

func (resscrptoken *bhResScriptToken) incActiveIO(ioRWs ...*bhio.BhIORW) *bhio.BhIORW {
	resscrptoken.activePreIOs()
	var atvPreIORWs []*bhio.BhIORW = []*bhio.BhIORW{}

	atvPreIORWs = append(atvPreIORWs, resscrptoken.atvPreIORWs...)

	if len(ioRWs) == 1 {
		atvPreIORWs = append(atvPreIORWs, ioRWs[0])
	} else {
		ioRW, _ := bhio.NewBhIORW()
		atvPreIORWs = append(atvPreIORWs, ioRW)
	}

	resscrptoken.atvPreIORWs = nil
	resscrptoken.atvPreIORWs = atvPreIORWs[:]
	atvPreIORWs = nil
	return resscrptoken.atvPreIORWs[len(resscrptoken.atvPreIORWs)-1]
}

func (resscrptoken *bhResScriptToken) activePreIOs() []*bhio.BhIORW {
	if resscrptoken.atvPreIORWs == nil {
		resscrptoken.atvPreIORWs = []*bhio.BhIORW{}
	}
	return resscrptoken.atvPreIORWs
}

func (ressrcptoken *bhResScriptToken) activeUnmatchedIO() *bhio.BhIORW {
	if ressrcptoken.atvUnmatchedIO == nil {
		ressrcptoken.atvUnmatchedIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.atvUnmatchedIO
}

func (ressrcptoken *bhResScriptToken) activeMatchedIO() *bhio.BhIORW {
	if ressrcptoken.atvMatchedIO == nil {
		ressrcptoken.atvMatchedIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.atvMatchedIO
}

//GENERAL

func (ressrcptoken *bhResScriptToken) generalIO() *bhio.BhIORW {
	if ressrcptoken.genIO == nil {
		ressrcptoken.genIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.genIO
}

//PASSIVE
func (ressrcptoken *bhResScriptToken) passiveAdhocIO() *bhio.BhIORW {
	if ressrcptoken.psvAdhocIO == nil {
		ressrcptoken.psvAdhocIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.psvAdhocIO
}

func (resscptoken *bhResScriptToken) passiveParkedAdhocIO() *bhio.BhIORW {
	if resscptoken.psvParkedAdhocIO == nil {
		resscptoken.psvParkedAdhocIO, _ = bhio.NewBhIORW()
	}
	return resscptoken.psvParkedAdhocIO
}

func (ressrcptoken *bhResScriptToken) passiveUnmatchedIO() *bhio.BhIORW {
	if ressrcptoken.psvUnmatchedIO == nil {
		ressrcptoken.psvUnmatchedIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.psvUnmatchedIO
}

func (ressrcptoken *bhResScriptToken) passiveFlushableIO() *bhio.BhIORW {
	if ressrcptoken.psvFlushableIO == nil {
		ressrcptoken.psvFlushableIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.psvFlushableIO
}

func (ressrcptoken *bhResScriptToken) passiveMatchedIO() *bhio.BhIORW {
	if ressrcptoken.psvMatchedIO == nil {
		ressrcptoken.psvMatchedIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.psvMatchedIO
}

func (ressrcptoken *bhResScriptToken) passiveValidMatchedIO() *bhio.BhIORW {
	if ressrcptoken.psvValidMatchIO == nil {
		ressrcptoken.psvValidMatchIO, _ = bhio.NewBhIORW()
	}
	return ressrcptoken.psvValidMatchIO
}

const activetoken int = 0
const passivetoken int = 1

func nextResScriptToken(bhresscrpt *BhResourceScript, prevToken *bhResScriptToken, iocur *bhio.BhReadWriteCursor, isjtext bool, path string, pathname string, pathext string, parked bool) (resscrptoken *bhResScriptToken) {
	if pathext == "" {
		pathext = filepath.Ext(path)
	}

	if pathname == "" {
		pathname = path[:len(path)-len(pathext)]
	}

	if pathext == "" {
		if prevToken == nil {
			pathext = filepath.Ext(bhresscrpt.res.path)
		} else {
			pathext = prevToken.pathext
		}
	}
	if pathext != "" && strings.LastIndex(path, "/") >= strings.LastIndex(path, ".") {
		path = path + pathext
	}

	resscrptoken = &bhResScriptToken{bhresscrpt: bhresscrpt, prevToken: prevToken, iocur: iocur, atvlabelsi: make([]int, 2), prvAtvb: 0, atvb: make([]byte, 1), psvlabelsi: make([]int, 2), prvPsvb: 0, psvb: make([]byte, 1), tokenstage: 0, path: path, pathext: pathext, pathname: pathname, isjtext: isjtext, psvElemFoundCount: 0}
	if strings.Index(pathname, "/") > -1 {
		resscrptoken.simplepathname = pathname[strings.LastIndex(pathname, "/")+1:]
	} else {
		resscrptoken.simplepathname = pathname
	}
	if isjtext {
		resscrptoken.atvlabels = []string{"[%", "%]"}
		resscrptoken.psvlabels = []string{"[", "]"}
	} else {
		resscrptoken.atvlabels = []string{"<%", "%>"}
		resscrptoken.psvlabels = []string{"<", ">"}
	}
	if parked {
		if prevToken != nil {
			if prevToken.psvElemFoundCount == 0 && prevToken.psvAdhocIO != nil && !prevToken.psvAdhocIO.Empty() {
				resscrptoken.passiveParkedAdhocIO().Print(prevToken.psvAdhocIO)
				prevToken.psvAdhocIO.Close()
			}
		}
	}
	return resscrptoken
}

func (ressrctoken *bhResScriptToken) parsed() (parsed bool, parseerr error) {
	//parsed = parseResScriptToken(ressrctoken, ressrctoken.prevToken)
	return parsed, parseerr
}

func (ressrctoken *bhResScriptToken) isParked() bool {
	return ressrctoken.psvElemFoundCount > 0
}

func parseResScriptToken(token *bhResScriptToken, prevtoken *bhResScriptToken, tknrb []byte) (isdone bool, isparked bool) {
	tknrn, tknrerr := tokenRead(token, tknrb)
	if token.tokenstage == activetoken {
		isdone = parseResTokenActiveStage(token, prevtoken, token.atvlabels, token.atvlabelsi, tknrn, tknrerr, tknrb)
	} else if token.tokenstage == passivetoken {
		isdone, isparked = parseResTokenPassiveStage(token, prevtoken, token.psvlabels, token.psvlabelsi, tknrn, tknrerr, tknrb)
	}
	return isdone, isparked
}

func tokenRead(ressrcptoken *bhResScriptToken, tknrb []byte) (tknrn int, tknrerr error) {
	switch ressrcptoken.tokenstage {
	case activetoken:
		for {
			if ressrcptoken.atvPreIORWs != nil && len(ressrcptoken.atvPreIORWs) > 0 {
				tknrn, tknrerr = ressrcptoken.atvPreIORWs[len(ressrcptoken.atvPreIORWs)-1].Read(tknrb)
				if tknrerr == io.EOF {
					if ressrcptoken.atvPreIORWs[len(ressrcptoken.atvPreIORWs)-1] == ressrcptoken.psvParkedAdhocIO {
						ressrcptoken.atvPreIORWs[len(ressrcptoken.atvPreIORWs)-1].Seek(0, 0)
					} else {
						ressrcptoken.atvPreIORWs[len(ressrcptoken.atvPreIORWs)-1].Close()
					}
					ressrcptoken.atvPreIORWs[len(ressrcptoken.atvPreIORWs)-1] = nil
					if len(ressrcptoken.atvPreIORWs) > 1 {
						ressrcptoken.atvPreIORWs = ressrcptoken.atvPreIORWs[0 : len(ressrcptoken.atvPreIORWs)-1]
					} else {
						ressrcptoken.atvPreIORWs = nil
					}
					continue
				}
				return tknrn, tknrerr
			}
			return ressrcptoken.iocur.Read(tknrb)
		}
	case passivetoken:
		if !ressrcptoken.atvUnmatchedIO.Empty() {
			tknrn, tknrerr = ressrcptoken.atvUnmatchedIO.Read(tknrb)
			if tknrerr == io.EOF {
				ressrcptoken.atvUnmatchedIO.Close()
			}
			return tknrn, tknrerr
		}
	}
	return tknrn, tknrerr
}

func parseResTokenActiveStage(token *bhResScriptToken, prevtoken *bhResScriptToken, labels []string, labelsi []int, tknrn int, tknrerr error, tknrb []byte) bool {
	if tknrerr == nil || tknrerr == io.EOF {
		if tknrn > 0 {
			if labelsi[1] == 0 && labelsi[0] < len(labels[0]) {
				if labelsi[0] > 1 && labels[0][labelsi[0]-1] == token.prvAtvb && labels[0][labelsi[0]] != tknrb[0] {
					token.activeUnmatchedIO().Print(labels[0:labelsi[0]])
					labelsi[0] = 0
					token.prvAtvb = tknrb[0]
				}
				if labels[0][labelsi[0]] == tknrb[0] {
					labelsi[0]++
					if labelsi[0] == len(labels[0]) {

						token.prvAtvb = 0
						return false
					} else {
						token.prvAtvb = tknrb[0]
						return false
					}
				} else {
					if labelsi[0] > 0 {
						token.activeUnmatchedIO().Print(labels[0][0:labelsi[0]])
						labelsi[0] = 0
					}
					token.prvAtvb = tknrb[0]
					token.activeUnmatchedIO().Print(string(token.prvAtvb))
					return false
				}
			} else if labelsi[0] == len(labels[0]) && labelsi[1] < len(labels[1]) {
				if labels[1][labelsi[1]] == tknrb[0] {
					labelsi[1]++
					if labelsi[1] == len(labels[1]) {

						labelsi[0] = 0
						labelsi[1] = 0
						token.prvAtvb = 0
						return false
					} else {
						return false
					}
				} else {
					//CHECK TO PARSE OUTSTANDING UNPARSED PASSIVE CONTENT
					if forwardActiveToPassive(token, prevtoken, tknrb...) {
						token.tokenstage = passivetoken
						return false
					}
					if !token.hasAtvMatched {
						token.hasAtvMatched = true
					}
					handlePassiveUnmatchedIO(token, prevtoken, false, token.isParked())
					handlePassiveFlushableIO(token, prevtoken, prevtoken != nil && prevtoken.hasAtvMatched, token.isParked())
					if labelsi[1] > 0 {
						token.activeMatchedIO().Print(labels[1][0:labelsi[1]])
						labelsi[1] = 0
					}
					if token.atvParkedMatchedIO != nil {
						if !token.atvParkedMatchedIO.Empty() {
							token.activeMatchedIO().Print(token.atvParkedMatchedIO)
							token.atvParkedMatchedIO.Close()
						}
					}
					token.activeMatchedIO().Print(tknrb)
					return false
				}
			}
		} else {
			//CHECK TO PARSE OUTSTANDING UNPARSED PASSIVE CONTENT
			if forwardActiveToPassive(token, prevtoken) {
				token.tokenstage = passivetoken
				return false
			}
			handlePassiveUnmatchedIO(token, prevtoken, false, token.isParked())
			/*if prevtoken == nil {
				handlePassiveFlushableIO(token, prevtoken, false, token.isParked())
			} else {
				handlePassiveFlushableIO(token, prevtoken, prevtoken.hasAtvMatched, token.isParked())
			}*/
			handlePassiveFlushableIO(token, prevtoken, prevtoken != nil && prevtoken.hasAtvMatched, token.isParked())
			flushcode(token.bhresscrpt, token, prevtoken, token.atvMatchedIO, token.isParked())
			return true
		}
	}
	return true
}

func forwardActiveToPassive(token *bhResScriptToken, prevToken *bhResScriptToken, parked ...byte) bool {
	if token.atvUnmatchedIO != nil {
		if !token.atvUnmatchedIO.Empty() {
			if len(parked) > 0 {
				if token.atvMatchedIO != nil {
					if !token.atvMatchedIO.Empty() {
						flushcode(token.bhresscrpt, token, prevToken, token.atvMatchedIO, token.isParked())
					}
				}
				if token.atvlabelsi[0] == len(token.atvlabels[0]) {
					token.incActiveIO().Print(token.atvlabels[0])
					token.atvlabelsi[0] = 0
					token.atvlabelsi[1] = 0
					token.activeIO().Print(parked)
				}
			}
			return true
		}
	}
	return false
}

func parseResTokenPassiveStage(token *bhResScriptToken, prevtoken *bhResScriptToken, labels []string, labelsi []int, tknrn int, tknrerr error, tknrb []byte) (isdone bool, isparked bool) {
	if tknrerr == nil || tknrerr == io.EOF {
		if tknrn > 0 {
			if labelsi[1] == 0 && labelsi[0] < len(labels[0]) {
				if labelsi[0] > 0 && labels[0][labelsi[0]-1] == token.prvPsvb && labels[0][labelsi[0]] != tknrb[0] {
					token.passiveUnmatchedIO().Print(labels[0][0:labelsi[0]])
					token.psvlabelsi[0] = 0
					token.prvPsvb = tknrb[0]
				}
				if labels[0][labelsi[0]] == tknrb[0] {
					labelsi[0]++
					if labelsi[0] == len(labels[0]) {

						return isdone, isparked
					} else {
						token.prvPsvb = tknrb[0]
						return isdone, isparked
					}
				} else {
					if token.atvMatchedIO != nil {
						if !token.atvMatchedIO.Empty() {
							if token.prevToken != nil {
								token.bhresscrpt.code().Print(token.atvMatchedIO)
							} else {
								token.bhresscrpt.code().Print(token.atvMatchedIO)
							}
							token.atvMatchedIO.Close()
						}
					}
					if labelsi[0] > 0 {
						token.passiveUnmatchedIO().Print(labels[0][0:labelsi[0]])
						token.psvlabelsi[0] = 0
					}
					token.prvPsvb = tknrb[0]
					token.passiveUnmatchedIO().Print(tknrb)
					return isdone, isparked
				}
			} else if labelsi[0] == len(labels[0]) && labelsi[1] < len(labels[1]) {
				if labels[1][labelsi[1]] == tknrb[0] {
					labelsi[1]++
					if labelsi[1] == len(labels[1]) {
						if valid, single, complexstart, complexend, parked, canIOAdhocContent, validres, validerr := validatePassiveMatchedIO(token, token.psvMatchedIO); valid && validerr == nil && (single || complexstart || complexend) {
							labelsi[0] = 0
							labelsi[1] = 0
							token.prvAtvb = 0
							token.psvlabelsi[0] = 0
							token.psvlabelsi[1] = 0
							token.prvPsvb = 0

							path := strings.Replace(strings.Replace(token.psvElemFound, ".:", "", -1), ":", "/", -1)
							pathext := filepath.Ext(path)
							pathname := ""

							if complexstart {
								handlePassiveUnmatchedIO(token, prevtoken, true, false)
								handlePassiveFlushableIO(token, prevtoken, prevtoken != nil && prevtoken.hasAtvMatched, false)
								if token.atvUnmatchedIO != nil && !token.atvUnmatchedIO.Empty() {
									token.incActiveIO().Print(token.atvUnmatchedIO.UnderlyingCursor())
									token.atvUnmatchedIO.Close()
									token.atvUnmatchedIO = nil
								}
							} else if single || complexend {
								handlePassiveUnmatchedIO(token, prevtoken, true, parked || token.isParked())
								handlePassiveFlushableIO(token, prevtoken, prevtoken != nil && prevtoken.hasAtvMatched, parked || token.isParked())
								if canIOAdhocContent {

									if token.psvParkedAdhocIO != nil && !token.psvParkedAdhocIO.Empty() {
										token.passiveUnmatchedIO().Print(token.psvParkedAdhocIO)
										//token.incActiveIO().Print(token.psvParkedAdhocIO)
										token.psvParkedAdhocIO.Seek(0, 0)
									}

									//if token.atvUnmatchedIO != nil && !token.atvUnmatchedIO.Empty() {

									//token.activeIO().Print(token.atvUnmatchedIO.UnderlyingCursor())
									//token.atvUnmatchedIO.Close()
									//}

									//s := token.activeIO().String()
									//fmt.Println(s)

									/*if token.atvUnmatchedIO != nil && !token.atvUnmatchedIO.Empty() {
										token.incActiveIO().Print(token.atvUnmatchedIO.UnderlyingCursor())
										token.atvUnmatchedIO.Close()
										if token.psvParkedAdhocIO != nil && !token.psvParkedAdhocIO.Empty() {
											token.activeIO().Print(token.psvParkedAdhocIO)
											token.psvParkedAdhocIO.Seek(0, 0)
										}
									} else {
										if token.psvParkedAdhocIO != nil && !token.psvParkedAdhocIO.Empty() {
											token.incActiveIO().Print(token.psvParkedAdhocIO)
											token.psvParkedAdhocIO.Seek(0, 0)
										}
									}*/

								} else {

									if validres != nil {
										iocur := validres.ioRW.ReadWriteCursor(true)
										token.bhresscrpt.parseToken(nextResScriptToken(token.bhresscrpt, token, iocur, token.isjtext, path, pathname, pathext, parked))
										if token.atvUnmatchedIO != nil && !token.atvUnmatchedIO.Empty() {
											token.incActiveIO().Print(token.atvUnmatchedIO.UnderlyingCursor())
											token.atvUnmatchedIO.Close()
											token.atvUnmatchedIO = nil
										}
										isparked = true
									} else if prevtoken != nil && single && token.psvElemFound == (".:"+token.simplepathname) {
										//FLUSH ADHOC HERE
										if prevtoken != nil && prevtoken.psvAdhocIO != nil && !prevtoken.psvAdhocIO.Empty() {
											fmt.Println(prevtoken.psvAdhocIO)
										}
									} else {
										if token.atvUnmatchedIO != nil && !token.atvUnmatchedIO.Empty() {
											token.incActiveIO().Print(token.atvUnmatchedIO.UnderlyingCursor())
											token.atvUnmatchedIO.Close()
											token.atvUnmatchedIO = nil
										}
									}
								}
							}
							if !canIOAdhocContent {
								token.tokenstage = activetoken
							}
							if token.psvMatchedIO != nil {
								token.psvMatchedIO.Close()
							}
						} else {
							if labelsi[0] > 0 {
								token.passiveUnmatchedIO().Print(labels[0][0:labelsi[0]])
								labelsi[0] = 0
							}
							if token.psvMatchedIO != nil {
								if !token.psvMatchedIO.Empty() {
									token.passiveUnmatchedIO().Print(token.psvMatchedIO)
									token.psvMatchedIO.Close()
								}
							}
							if labelsi[1] > 0 {
								token.passiveUnmatchedIO().Print(labels[1][0:labelsi[1]])
								labelsi[1] = 0
							}
						}

						token.prvPsvb = 0
						return isdone, isparked
					} else {
						return isdone, isparked
					}
				} else {
					handlePassiveUnmatchedIO(token, prevtoken, false, token.isParked())
					if labelsi[1] > 0 {
						token.passiveMatchedIO().Print(labels[1][0:labelsi[1]])
						labelsi[1] = 0
					}
					token.passiveMatchedIO().Print(tknrb)
					return isdone, isparked
				}
			}
		} else {
			if labelsi[0] > 0 {
				token.passiveUnmatchedIO().Print(labels[0][0:labelsi[0]])
				labelsi[0] = 0
			}
			if token.psvMatchedIO != nil {
				if !token.psvMatchedIO.Empty() {
					token.passiveUnmatchedIO().Print(token.psvMatchedIO)
					token.psvMatchedIO.Close()
				}
			}
			if labelsi[1] > 0 {
				token.passiveUnmatchedIO().Print(labels[1][0:labelsi[1]])
				labelsi[1] = 0
			}

			token.prvPsvb = 0
			handlePassiveUnmatchedIO(token, prevtoken, false, token.isParked())
			token.tokenstage = activetoken
			if tknrerr == nil || tknrerr == io.EOF {
				return isdone, isparked
			} else {
				isdone = true
				return isdone, isparked
			}
		}
	}
	return isdone, isparked
}

func isValidStartTagElemName(startTagNameToTest string) bool {
	return false
}

type validateStage int

const (
	validatenone        validateStage = 0
	validateproperty    validateStage = 1
	validateassign      validateStage = 2
	validatepropval     validateStage = 3
	validatecapturedval validateStage = 4
)

func isValidPassiveElemName(cmplexend bool, single bool, token *bhResScriptToken, elemNameToTest string) bool {
	if token.psvElemFoundCount > 0 {
		return (cmplexend || !(cmplexend || single)) && token.psvElemFound == elemNameToTest
	}
	return true
}

func isValidPassiveElemProperty(token *bhResScriptToken, elemPropToTest string) bool {
	return false
}

func isValidNumericValue(token *bhResScriptToken, numvalToTest string) bool {
	return false
}

func validatePassiveMatchedIO(token *bhResScriptToken, r *bhio.BhIORW) (valid bool, single bool, complexstart bool, complexend bool, parked bool, canIOAdhocContent bool, validres *BhResource, validerr error) {
	valid = false
	if token.psvValidMatchIO != nil {
		token.psvValidMatchIO.Close()
		token.psvValidMatchIO = nil
	}
	canValidate := true
	single = false
	complexend = false
	complexstart = false
	foundBackspace := false
	foundAssign := false
	textpar := uint8(0)
	foundTextVal := false
	foundValStart := false
	prevtextc := uint8(0)
	elemFound := ""
	propFound := ""
	if r != nil {
		r.Seek(0, 0)
		vr := make([]byte, 1)
		validstage := validatenone
		for canValidate {
			if vrn, vrnerr := r.Read(vr); vrnerr == nil && vrn != 0 {
				if validstage == validatenone {
					if strings.TrimSpace(string(vr)) == "" || string(vr) == "/" {
						if string(vr) == "/" {
							if !foundBackspace {
								foundBackspace = true
								if complexend = (token.psvValidMatchIO == nil || token.psvValidMatchIO.Empty()); !complexend {
									single = (token.psvValidMatchIO == nil || !token.psvValidMatchIO.Empty())
								}
							} else {
								valid = false
								canValidate = false
							}
							continue
						}
						if token.psvValidMatchIO != nil {

							if regexptagstart.MatchString(token.psvValidMatchIO.String()) {
								if elemFound = token.psvValidMatchIO.String(); isValidPassiveElemName(complexend, single, token, elemFound) {
									if token.psvValidMatchIO != nil {
										token.psvValidMatchIO.Close()
										token.psvValidMatchIO = nil
									}
									propFound = ""
									validstage = validateproperty
								} else {
									valid = false
									canValidate = false
								}
							}

						}
					} else {
						token.passiveValidMatchedIO().Write(vr)
					}
				} else if validstage == validatecapturedval || validstage == validateproperty {
					if validstage == validatecapturedval {
						if propFound != "" {
							propFound = ""
						}
						if strings.TrimSpace(string(vr)) == "" || string(vr) == "=" {
							if string(vr) == "=" {
								if !foundAssign {
									if propFound == "" {
										if token.psvValidMatchIO == nil || token.psvValidMatchIO.Empty() {
											valid = false
											canValidate = false
											continue
										} else {
											foundAssign = true
										}
									} else if propFound != "" {
										if token.psvValidMatchIO != nil {
											token.psvValidMatchIO.Close()
											token.psvValidMatchIO = nil
										}
										if foundAssign {
											validstage = validateassign
										}
										continue
									}
								} else {
									valid = false
									canValidate = false
									continue
								}
							}
							if regexprop.MatchString(token.psvValidMatchIO.String()) {
								if token.psvValidMatchIO != nil && isValidPassiveElemProperty(token, token.psvValidMatchIO.String()) {
									propFound = token.psvValidMatchIO.String()
									if token.psvValidMatchIO != nil {
										token.psvValidMatchIO.Close()
										token.psvValidMatchIO = nil
									}
									if foundAssign {
										validstage = validateassign
									}
								} else {
									valid = false
									canValidate = false
									continue
								}
							} else {
								valid = false
								canValidate = false
								continue
							}
						} else {
							token.passiveValidMatchedIO().Write(vr)
						}
					}
				} else if validstage == validateassign || validstage == validatepropval {
					if validstage == validateassign {
						validstage = validatepropval
					}
					if !foundValStart {
						if strings.TrimSpace(string(vr)) != "" {
							if textpar == 0 && (string(vr) == "\"" || string(vr) == "'") {
								foundTextVal = true
								textpar = string(vr)[0]
								foundValStart = true
								prevtextc = 0
							} else if string(vr) == "/" {
								if foundBackspace {
									canValidate = false
									valid = false
								} else {
									//TODO CAPTURE property
									if token.psvValidMatchIO == nil || token.psvValidMatchIO.Empty() {
										token.possibleProperties()[propFound] = ""
									} else {
										token.possibleProperties()[propFound] = token.psvValidMatchIO.String()
									}
									if token.psvValidMatchIO != nil {
										token.psvValidMatchIO.Close()
										token.psvValidMatchIO = nil
									}
									validstage = validatecapturedval
									foundBackspace = true
									single = true
								}
							} else {
								foundValStart = true
							}
						}
					} else {
						if foundTextVal {
							if prevtextc == textpar && string(vr)[0] == textpar {
								token.passiveValidMatchedIO().Write(vr)
								prevtextc = string(vr)[0]
							} else if prevtextc != textpar && string(vr)[0] == textpar {
								//TODO CAPTURE property
								textpar = uint8(0)
								prevtextc = uint8(0)
								foundTextVal = false
								if token.psvValidMatchIO == nil || token.psvValidMatchIO.Empty() {
									token.possibleProperties()[propFound] = ""
								} else {
									token.possibleProperties()[propFound] = token.psvValidMatchIO.String()
								}
								if token.psvValidMatchIO != nil {
									token.psvValidMatchIO.Close()
									token.psvValidMatchIO = nil
								}
								validstage = validatecapturedval
							} else {
								token.passiveValidMatchedIO().Write(vr)
							}
						} else {
							if strings.TrimSpace(string(vr)) == "" || string(vr) == "/" {
								if string(vr) == "/" {
									if foundBackspace {
										canValidate = false
										valid = false
									} else {
										foundBackspace = true
										single = true
										foundValStart = false
									}
								}
								if token.psvValidMatchIO != nil {
									if token.psvValidMatchIO.Empty() {
										//TODO CAPTURE EMPTY VAL
										if token.psvValidMatchIO == nil || token.psvValidMatchIO.Empty() {
											token.possibleProperties()[propFound] = ""
										} else {
											token.possibleProperties()[propFound] = token.psvValidMatchIO.String()
										}
										if token.psvValidMatchIO != nil {
											token.psvValidMatchIO.Close()
											token.psvValidMatchIO = nil
										}
										validstage = validatecapturedval
										foundValStart = false
									} else {
										if truefalse := token.psvValidMatchIO.String(); truefalse == "true" || truefalse == "false" {
											//TODO CAPTURE BOOLEAN VAL
											token.possibleProperties()[propFound] = truefalse
											if token.psvValidMatchIO != nil {
												token.psvValidMatchIO.Close()
												token.psvValidMatchIO = nil
											}
											validstage = validatecapturedval
											foundValStart = false
										} else if numval := token.psvValidMatchIO.String(); numval == "true" || numval == "false" || isValidNumericValue(token, numval) {
											//TODO CAPTURE NUMBER VAL
											if numval == "true" {
												token.possibleProperties()[propFound] = "1"
											} else if numval == "false" {
												token.possibleProperties()[propFound] = "0"
											} else {
												token.possibleProperties()[propFound] = numval
											}
											if token.psvValidMatchIO != nil {
												token.psvValidMatchIO.Close()
												token.psvValidMatchIO = nil
											}
											validstage = validatepropval
											foundValStart = false
										} else {
											canValidate = false
											valid = false
										}
									}
								}
							} else {
								token.passiveValidMatchedIO().Write(vr)
							}
						}
					}
				} else {
					token.passiveValidMatchedIO().Write(vr)
				}
			} else {
				if vrnerr != nil && vrnerr != io.EOF {
					validerr = vrnerr
				}
				break
			}
		}
		if canValidate {
			switch validstage {
			case validatenone:
				if token.psvValidMatchIO != nil {
					if regexptagstart.MatchString(token.psvValidMatchIO.String()) {
						if elemFound = token.psvValidMatchIO.String(); isValidPassiveElemName(complexend, single, token, elemFound) {
							if token.psvValidMatchIO != nil {
								token.psvValidMatchIO.Close()
								token.psvValidMatchIO = nil
							}
							propFound = ""
							validstage = validateproperty
							valid = true
						} else {
							valid = false
							canValidate = false
						}
					} else {
						valid = false
						canValidate = false
					}
				}
				break
			default:
				break
			}
			if valid {
				if complexend && token.psvElemFoundCount > 0 {
					token.psvElemFoundCount--
					if valid = token.psvElemFoundCount == 0; valid {
						parked = true
					}
				} else if single {
					valid = token.psvElemFoundCount == 0
				} else {
					complexstart = !single && !complexend
				}
			}
		}
		r.Seek(0, 0)
	}
	if valid {
		pathToTest := strings.Replace(strings.Replace(elemFound, ".:", "", 1), ":", "/", -1)
		if pext := filepath.Ext(pathToTest); pext == "" {
			if token.prevToken == nil {
				pext = filepath.Ext(token.bhresscrpt.res.path)
			} else {
				pext = token.pathext
			}
			pathToTest = pathToTest + pext
		}
		if single {
			if elemFound == (".:" + token.simplepathname) {
				if token.prevToken == nil {
					valid = false
				} else {
					canIOAdhocContent = true
				}
			}
		}
		if valid {
			if !(single && elemFound == (".:"+token.simplepathname)) {
				if validres = RegisteredResource(token.bhresscrpt.res.root, pathToTest); validres == nil {
					if validres = registerResourceFromRootPath(token.bhresscrpt.res.root, pathToTest); validres == nil {
						valid = false
					} else if complexstart {
						validres = nil
						valid = token.psvElemFoundCount == 0
						token.psvElemFoundCount++
					}
				} else if complexstart {
					validres = nil
					valid = token.psvElemFoundCount == 0
					token.psvElemFoundCount++
				}
			}
		}
	}
	if !valid {
		token.clearPassiveProperties()
	} else {
		token.psvElemFound = elemFound
	}
	return valid, single, complexstart, complexend, parked, canIOAdhocContent, validres, validerr
}

func (token *bhResScriptToken) clearPassiveProperties() {
	if len(token.psvProperties) > 0 {
		for p, _ := range token.psvProperties {
			delete(token.psvProperties, p)
		}
	}
}

func handlePassiveUnmatchedIO(token *bhResScriptToken, prevToken *bhResScriptToken, forcelushCode bool, parked bool) {
	if prevToken != nil || forcelushCode {
		if token.atvMatchedIO != nil && !token.atvMatchedIO.Empty() && token.tokenstage == passivetoken {
			flushcode(token.bhresscrpt, token, prevToken, token.atvMatchedIO, parked)
		}
	}
	if token.psvUnmatchedIO != nil {
		if !token.psvUnmatchedIO.Empty() {
			token.passiveFlushableIO().Print(token.psvUnmatchedIO)
			token.psvUnmatchedIO.Close()
		}
	}
}

func handlePassiveFlushableIO(token *bhResScriptToken, prevToken *bhResScriptToken, forceMatchedIO bool, parked bool) {
	if token.psvFlushableIO != nil {
		if !token.psvFlushableIO.Empty() {
			flushcontent(token.bhresscrpt, token, prevToken, forceMatchedIO, token.psvFlushableIO, parked)
			token.psvFlushableIO.Close()
		}
	}
}

/*func handleAtvUnmatchedIO(token *bhResScriptToken, prevToken *bhResScriptToken, parked ...byte) bool {
	if token.atvUnmatchedIO != nil {
		if !token.atvUnmatchedIO.Empty() {
			if len(parked) > 0 {
				if token.atvMatchedIO != nil {
					if !token.atvMatchedIO.Empty() {
						flushcode(token.bhresscrpt, token, prevToken, token.atvMatchedIO)
					}
				}
				if token.atvlabelsi[0] == len(token.atvlabels[0]) {
					if token.psvElemFoundCount == 0 {
						token.activeParkedUnmatchedIO().Print(token.atvlabels[0])
					}
					token.atvlabelsi[0] = 0
					token.atvlabelsi[1] = 0
					token.activeParkedUnmatchedIO().Print(parked)
				}
			}
			token.tokenstage = passivetoken
			return true
		}
	}
	return false
}*/

func (resscrptoken *bhResScriptToken) cleanupBhResScriptToken() {

	if resscrptoken.genIO != nil {
		if !resscrptoken.genIO.Empty() {
			if resscrptoken.prevToken != nil {
				if resscrptoken.bhresscrpt != nil {
					resscrptoken.prevToken.incActiveIO().Print(resscrptoken.genIO)
				}
			}
			resscrptoken.genIO.Close()
			resscrptoken.genIO = nil
		}
	}

	if resscrptoken.prevToken != nil {
		if resscrptoken.bhresscrpt != nil {
			resscrptoken.bhresscrpt.parseToken(resscrptoken.prevToken)
			resscrptoken.bhresscrpt = nil
		}
		resscrptoken.prevToken = nil
	}
	if resscrptoken.bhresscrpt != nil {
		resscrptoken.bhresscrpt = nil
	}
	if resscrptoken.atvlabels != nil {
		resscrptoken.atvlabels = nil
	}
	if resscrptoken.atvlabelsi != nil {
		resscrptoken.atvlabelsi = nil
	}
	if resscrptoken.prvAtvb != 0 {
		resscrptoken.prvAtvb = 0
	}

	if resscrptoken.atvMatchedIO != nil {
		resscrptoken.atvMatchedIO.Close()
		resscrptoken.atvMatchedIO = nil
	}

	if resscrptoken.atvParkedMatchedIO != nil {
		resscrptoken.atvParkedMatchedIO.Close()
		resscrptoken.atvParkedMatchedIO = nil
	}

	/*if resscrptoken.atvParkedUnmatchedIO != nil {
		resscrptoken.atvParkedUnmatchedIO.Close()
		resscrptoken.atvParkedUnmatchedIO = nil
	}*/

	if resscrptoken.atvUnmatchedIO != nil {
		resscrptoken.atvUnmatchedIO.Close()
		resscrptoken.atvUnmatchedIO = nil
	}

	if resscrptoken.atvPreIORWs != nil {
		for len(resscrptoken.atvPreIORWs) > 0 {
			resscrptoken.atvPreIORWs[0].Close()
			resscrptoken.atvPreIORWs[0] = nil
			if len(resscrptoken.atvPreIORWs) > 1 {
				resscrptoken.atvPreIORWs = resscrptoken.atvPreIORWs[1:]
			} else {
				break
			}
		}
		resscrptoken.atvPreIORWs = nil
	}

	if resscrptoken.psvlabels != nil {
		resscrptoken.psvlabels = nil
	}
	if resscrptoken.psvlabelsi != nil {
		resscrptoken.psvlabelsi = nil
	}
	if resscrptoken.prvPsvb != 0 {
		resscrptoken.prvPsvb = 0
	}

	if resscrptoken.psvFlushableIO != nil {
		resscrptoken.psvFlushableIO.Close()
		resscrptoken.psvFlushableIO = nil
	}

	if resscrptoken.psvMatchedIO != nil {
		resscrptoken.psvMatchedIO.Close()
		resscrptoken.psvMatchedIO = nil
	}

	if resscrptoken.psvParkedMatchedIO != nil {
		resscrptoken.psvParkedMatchedIO.Close()
		resscrptoken.psvParkedMatchedIO = nil
	}

	if resscrptoken.psvUnmatchedIO != nil {
		resscrptoken.psvUnmatchedIO.Close()
		resscrptoken.psvUnmatchedIO = nil
	}

	if resscrptoken.psvAdhocIO != nil {
		resscrptoken.psvAdhocIO.Close()
		resscrptoken.psvAdhocIO = nil
	}

	if resscrptoken.psvParkedAdhocIO != nil {
		resscrptoken.psvParkedAdhocIO.Close()
		resscrptoken.psvParkedAdhocIO = nil
	}

	if resscrptoken.iocur != nil {
		resscrptoken.iocur.Close()
		resscrptoken.iocur = nil
	}

	if resscrptoken.psvProperties != nil {
		resscrptoken.clearPassiveProperties()
		resscrptoken.psvProperties = nil
	}
}

func (resscrpt *BhResourceScript) cleanupBhResourceScript() {
	if resscrpt.cde != nil {
		resscrpt.cde.Close()
		resscrpt.cde = nil
	}
	if resscrpt.cntnt != nil {
		resscrpt.cntnt.Close()
		resscrpt.cntnt = nil
	}
	if resscrpt.prg != nil {
		resscrpt.prg = nil
	}
}

func (resscrpt *BhResourceScript) parseToken(tokenToParse *bhResScriptToken) {
	done := make(chan bool, 1)
	go func() {
		done <- true
		resscrpt.parseTokensChannel <- tokenToParse
	}()
	<-done
	close(done)
}

func processParsableTokens(done chan bool, resscrpt *BhResourceScript, token *bhResScriptToken) {
	doneParsing := make(chan bool)
	tknrb := make([]byte, 1)
	for {
		if token == nil {
			break
		} else {
			go func() {
				continueit := true
				for continueit {
					if isdone, isparked := parseResScriptToken(token, token.prevToken, tknrb); isdone || isparked {
						if isdone {
							token.cleanupBhResScriptToken()
							token = nil
						}
						if isparked {
							select {
							case token = <-resscrpt.parseTokensChannel:
								if token != nil {
									continue
								} else {
									continueit = false
									continue
								}
							default:
							}
						} else {
							select {
							case token = <-resscrpt.parseTokensChannel:
							default:
							}
							if token != nil {
								continue
							} else {
								continueit = false
								continue
							}
						}
					}
				}
				doneParsing <- true
			}()
			<-doneParsing
		}
	}
	done <- true
}

func (resscrpt *BhResourceScript) loadScript(isjsext bool) (err error) {
	resscrpt.cleanupBhResourceScript()

	iocur := resscrpt.res.ioRW.ReadWriteCursor(true)
	resscrpt.parseTokensChannel = make(chan *bhResScriptToken)

	done := make(chan bool)

	go processParsableTokens(done, resscrpt, nextResScriptToken(resscrpt, nil, iocur, isjsext, resscrpt.res.path, "", "", false))

	/*go func() {
		var resscrptkn *bhResScriptToken = nextResScriptToken(resscrpt, nil, iocur, isjsext, resscrpt.res.path, "", "")
		for {
			select {
			case resscrptkn = <-resscrpt.parseTokensChannel:
				break
			default:
				break
			}
			if resscrptkn == nil {
				break
			} else {
				if parsed, parsederr := resscrptkn.parsed(); parsed && parsederr == nil {
					resscrptkn.cleanupBhResScriptToken()
					resscrptkn = nil
				} else if parsederr != nil {
					resscrptkn.cleanupBhResScriptToken()
					err = parsederr
					break
				}
			}
		}
		done <- true
	}()*/
	<-done
	close(done)
	iocur.Close()
	close(resscrpt.parseTokensChannel)
	iocur = nil
	if err != nil {
		resscrpt.cleanupBhResourceScript()
	} else if resscrpt.cde != nil && !resscrpt.cde.Empty() {
		//fmt.Println(resscrpt.cde)
		//fmt.Println()
		if prg, prgerr := goja.Compile("", resscrpt.cde.String(), false); prgerr == nil {
			resscrpt.prg = prg
		} else {
			resscrpt.prgerr = prgerr
		}
	}
	if resscrpt.cntnt != nil && !resscrpt.cntnt.Empty() {
		//fmt.Println(resscrpt.cntnt)
	}
	return err
}

func (resscrpt *BhResourceScript) hasCode() bool {
	return resscrpt.cde != nil && !resscrpt.cde.Empty()
}

func (resscrpt *BhResourceScript) codeOnly() bool {
	return resscrpt.hasCode() && !resscrpt.hasContent()
}

func (resscrpt *BhResourceScript) contentOnly() bool {
	return resscrpt.hasContent() && !resscrpt.hasCode()
}

func (resscrpt *BhResourceScript) hasContent() bool {
	return resscrpt.cntnt != nil && !resscrpt.cntnt.Empty()
}

func (resscrpt *BhResourceScript) empty() bool {
	return !resscrpt.hasCode() && !resscrpt.hasContent()
}

/*func tokenActiveReader(ressrcptoken *bhResScriptToken) io.ReadSeeker {
	switch ressrcptoken.tokenstage {
	case activetoken:
		if ressrcptoken.atvParkedUnmatchedIO != nil && !ressrcptoken.atvParkedUnmatchedIO.Empty() {
			if ressrcptoken.atvParkedUnmatchedIO.Size() == ressrcptoken.atvParkedUnmatchedIO.SeekIndex() {
				ressrcptoken.atvParkedUnmatchedIO.Close()
				if ressrcptoken.atvPreIORW != nil && !ressrcptoken.atvPreIORW.Empty() {
					if ressrcptoken.atvPreIORW.Size() == ressrcptoken.atvPreIORW.SeekIndex() {
						ressrcptoken.atvPreIORW.Close()
					} else {
						return ressrcptoken.atvPreIORW
					}
				}
				return ressrcptoken.iocur
			} else {
				return ressrcptoken.atvParkedUnmatchedIO
			}
		} else {
			if ressrcptoken.atvPreIORW != nil && !ressrcptoken.atvPreIORW.Empty() {
				if ressrcptoken.atvPreIORW.Size() == ressrcptoken.atvPreIORW.SeekIndex() {
					ressrcptoken.atvPreIORW.Close()
				} else {
					return ressrcptoken.atvPreIORW
				}
			}
			return ressrcptoken.iocur
		}

	case passivetoken:
		return ressrcptoken.atvUnmatchedIO
	}
	return nil
}*/
