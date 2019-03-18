package bhscript

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dop251/goja"
)

//BhScript BhScript
type BhScript struct {
	vm        *goja.Runtime
	boundobjs map[string]interface{}
}

func (scp *BhScript) CleanupBhScript() {
	if scp.boundobjs != nil {
		for name, _ := range scp.boundobjs {
			scp.Unbound(name)
		}
		scp.boundobjs = nil
	}
	if scp.vm != nil {
		scp.vm = nil
	}
}

func NewBhSrcipt() *BhScript {
	bhscp := &BhScript{vm: goja.New(), boundobjs: make(map[string]interface{})}
	return bhscp
}

func (scp *BhScript) Bound(name string, obj interface{}) {
	if bobj, ok := scp.boundobjs[name]; ok {
		if bobj != obj {
			scp.Unbound(name)
			scp.boundobjs[name] = obj
		}
	} else {
		scp.boundobjs[name] = obj
	}
	scp.vm.Set(name, obj)
}

func (scp *BhScript) Unbound(name string) {
	if _, ok := scp.boundobjs[name]; ok {
		obj := scp.boundobjs[name]
		delete(scp.boundobjs, name)
		scp.deleteObj(obj)
		obj = nil
	}
}

func (scp *BhScript) deleteObj(obj interface{}) {
	obj = nil
}

func (scp *BhScript) ScriptedFunc(call goja.Value, params ...interface{}) (v goja.Value) {

	if f, ok := goja.AssertFunction(call); ok {
		if len(params) > 0 {
			pvalues := make([]goja.Value, len(params))
			for n, p := range params {
				pvalues[n] = scp.vm.ToValue(p)
			}
			if res, err := f(nil, pvalues...); err == nil {
				v = res
			} else {
				fmt.Println(err.Error())
			}
		} else {
			if res, err := f(nil); err == nil {
				v = res
			} else {
				fmt.Println(err.Error())
			}
		}
	}
	return v
}

func CallScriptFunc(scp *BhScript, call goja.FunctionCall, params ...interface{}) (v goja.Value) {
	if f, ok := goja.AssertFunction(scp.vm.ToValue(call)); ok {
		if len(params) > 0 {
			pvalues := make([]goja.Value, len(params)+1)

			for n, p := range params {
				if n == 0 {

				} else {
					pvalues[n] = scp.vm.ToValue(p)
				}
			}
			if res, err := f(nil, pvalues...); err == nil {
				v = res
			} else {
				fmt.Println(err.Error())
			}
		} else {
			if res, err := f(nil); err == nil {
				v = res
			} else {
				fmt.Println(err.Error())
			}
		}
	}
	return v
}

func (scp *BhScript) Eval(src interface{}) (evalerr error) {
	if prg, prgok := src.(*goja.Program); prgok {
		_, prgerr := scp.vm.RunProgram(prg)
		return prgerr
	}
	var source string

	if str, strok := src.(string); strok {
		source = str
	} else if ior, iorok := src.(io.Reader); iorok {
		var bfr bytes.Buffer
		if _, err := io.Copy(&bfr, ior); err == nil {
			source = string(bfr.Bytes())
		}
	}

	p, err := goja.Compile("", source, false)
	if err != nil {
		evalerr = err
	}
	_, evalerr = scp.vm.RunProgram(p)
	return evalerr
}
