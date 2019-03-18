package bhservlet

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/EvertJoubert/bhind/bhio"
)

//WidgetIntercace interface implemented by all Widgets
type WidgetInterface interface {
	io.ReadSeeker
	InvokeWidget()
	LoadWidget()
	ProcessWidget()
	UnloadWidget()
	Servlet() *BhServletRequest
	DisposeWidget()
	InvokeEvent(...string)
	Out() *bhio.BhIORW
	In() *bhio.BhReadWriteCursor
	Name() string
	Path() string
	PathName() string
	Extension() string
	FullPathName() string
}

type BhWidget struct {
	srvlt              *BhServletRequest
	content            *bhio.BhIORW
	contentcur         *bhio.BhReadWriteCursor
	processWidgetEvent ProcessWidgetEvent
	namedEvents        map[string]func(WidgetInterface)
	path               string
	name               string
	ext                string
}

func (bhwdgt *BhWidget) Name() string {
	return bhwdgt.name
}

func (bhwdgt *BhWidget) Path() string {
	return bhwdgt.path
}

func (bhwdgt *BhWidget) PathName() string {
	return bhwdgt.path + bhwdgt.name
}

func (bhwdgt *BhWidget) Extension() string {
	return bhwdgt.ext
}

func (bhwdgt *BhWidget) FullPathName() string {
	return bhwdgt.path + bhwdgt.name + bhwdgt.ext
}

func (bhwdgt *BhWidget) InvokeEvent(event ...string) {
	if len(bhwdgt.namedEvents) > 0 && len(event) > 0 {
		for _, evToFind := range event {
			if ev, evko := bhwdgt.namedEvents[evToFind]; evko {
				ev(bhwdgt)
			}
		}
	}
}

func (bhwdgt *BhWidget) InvokeWidget() {
	if bhwdgt.content != nil {
		bhwdgt.content.Close()
		bhwdgt.content = nil
	}
	if bhwdgt.contentcur != nil {
		bhwdgt.contentcur.Close()
		bhwdgt.contentcur = nil
	}
	if bhwdgt.srvlt.httpw != nil {
		bhwdgt.content, _ = bhio.NewBhIORW(bhwdgt.srvlt.httpw)
	} else if bhwdgt.srvlt.altw != nil {
		bhwdgt.content, _ = bhio.NewBhIORW(bhwdgt.srvlt.altw)
	}
	if bhwdgt.content != nil {
		bhwdgt.contentcur = bhwdgt.content.ReadWriteCursor(true)
	}
}

func (bhwdgt *BhWidget) LoadWidget() {
}

type ProcessWidgetEvent = func(WidgetInterface)

func (bhwdgt *BhWidget) ProcessWidget() {
	if possibleevents := bhwdgt.Servlet().Parameters().Parameter(bhwdgt.Name() + "-event"); len(possibleevents) > 0 {
		var invokeevent func(string) = func(event string) {
			bhwdgt.InvokeEvent(event)
		}
		for _, pevent := range possibleevents {
			if strings.Index(pevent, "|") == -1 {
				invokeevent(pevent)
			} else {
				for _, pev := range strings.Split(pevent, "|") {
					invokeevent(pev)
				}
			}
		}
		invokeevent = nil
	} else {
		if bhwdgt.processWidgetEvent != nil {
			bhwdgt.processWidgetEvent(bhwdgt)
		}
	}
}

func (bhwdgt *BhWidget) UnloadWidget() {
}

func (bhwdgt *BhWidget) DisposeWidget() {
	if bhwdgt.content != nil {
		bhwdgt.content.Close()
		bhwdgt.content = nil
	}
	if bhwdgt.contentcur != nil {
		bhwdgt.contentcur.Close()
		bhwdgt.contentcur = nil
	}
	if bhwdgt.srvlt != nil {
		bhwdgt.srvlt = nil
	}
	if bhwdgt.processWidgetEvent != nil {
		bhwdgt.processWidgetEvent = nil
	}
	if bhwdgt.namedEvents != nil {
		if len(bhwdgt.namedEvents) > 0 {
			for name, _ := range bhwdgt.namedEvents {
				bhwdgt.namedEvents[name] = nil
				delete(bhwdgt.namedEvents, name)
			}
		}
		bhwdgt.namedEvents = nil
	}
}

func (bhwdgt *BhWidget) In() *bhio.BhReadWriteCursor {
	return bhwdgt.contentcur
}

func (bhwdgt *BhWidget) Out() *bhio.BhIORW {
	return bhwdgt.content
}

func (bhwdgt *BhWidget) Servlet() *BhServletRequest {
	return bhwdgt.srvlt
}

func (bhwdgt *BhWidget) Read(p []byte) (n int, err error) {
	if bhwdgt.contentcur != nil {
		n, err = bhwdgt.contentcur.Read(p)
	}
	return n, err
}

func (bhwdgt *BhWidget) Seek(offset int64, whence int) (n int64, err error) {
	if bhwdgt.contentcur != nil {
		n, err = bhwdgt.contentcur.Seek(offset, whence)
	}
	return n, err
}

func NewBhWidget(path string, srvlt *BhServletRequest, events ...interface{}) (bhwgdt *BhWidget) {
	ext := filepath.Ext(path)
	if ext != "" {
		path = path[:len(path)-len(ext)]
	}
	name := path
	if strings.Index(name, "/") > -1 {
		name = path[strings.LastIndex(path, "/")+1:]
		path = path[:strings.LastIndex(path, "/")+1]
	} else {
		path = "/"
	}

	bhwgdt = &BhWidget{srvlt: srvlt, ext: ext, path: path, name: name}
	var invokeEventInterface func(interface{}) = func(e interface{}) {
		if processev, processevok := e.(ProcessWidgetEvent); processevok {
			bhwgdt.processWidgetEvent = processev
		} else if nmdevents, nmdeventsok := e.(map[string]func(WidgetInterface)); nmdeventsok {
			for name, event := range nmdevents {
				if bhwgdt.namedEvents == nil {
					bhwgdt.namedEvents = make(map[string]func(WidgetInterface))
				}
				bhwgdt.namedEvents[name] = event
			}
		}
	}
	if len(events) > 0 {
		for _, event := range events {
			if a, aok := event.([]interface{}); aok {
				if len(a) > 0 {
					for _, ai := range a {
						invokeEventInterface(ai)
					}
				}
			} else {
				invokeEventInterface(event)
			}
		}
	}

	return bhwgdt
}
