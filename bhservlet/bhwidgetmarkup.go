package bhservlet

import (
	"strings"

	"github.com/dop251/goja"
)

func (bhwdgt *BhWidget) StartElem(elemname string, props ...interface{}) {
	bhwdgt.Out().Print("<", elemname)
	bhwdgt.printProps(props...)
	bhwdgt.Out().Print(">")
}

func (bhwdgt *BhWidget) printProps(props ...interface{}) {
	if len(props) > 0 {
		var ouputProp func(string, string) = func(pname string, pvalue string) {
			bhwdgt.Out().Print(" ", pname, "=", "\"", pvalue, "\"")
		}
		for _, prop := range props {
			if sprop, spropok := prop.(string); spropok && strings.Index(sprop, "=") > 0 {
				ouputProp(sprop[:strings.Index(sprop, "=")], sprop[strings.Index(sprop, "=")+1:])
			} else if mprop, mpropok := prop.(map[string]string); mpropok && len(mprop) > 0 {
				for pname, pvalue := range mprop {
					ouputProp(pname, pvalue)
				}
			} else if pprops, ppropsok := prop.([]string); ppropsok && len(props) > 0 {
				for _, prop := range pprops {
					if strings.Index(prop, "=") > 0 {
						ouputProp(prop[:strings.Index(prop, "=")], prop[strings.Index(prop, "=")+1:])
					}
				}
			}
		}
	}
}

func (bhwdgt *BhWidget) Element(elemname string, elemprops ...interface{}) {
	elmprops, elmfuncs := splitPropsIntoFuncAndProps(elemprops...)
	bhwdgt.StartElem(elemname, elmprops...)
	callElemFuncs(bhwdgt, elmfuncs)
	bhwdgt.EndElem(elemname)
}

func callElemFuncs(bhwdgt *BhWidget, elemfuncs []func(*BhWidget, ...interface{}), args ...interface{}) {
	if len(elemfuncs) > 0 {
		for _, elmfunc := range elemfuncs {
			elmfunc(bhwdgt, args...)
		}
	}
}

func splitPropsIntoFuncAndProps(propstosplit ...interface{}) (props []interface{}, funcs []func(*BhWidget, ...interface{})) {
	if len(propstosplit) > 0 {
		for _, propToTest := range propstosplit {
			if f, fok := propToTest.(func(*BhWidget, ...interface{})); fok {
				if funcs == nil {
					funcs = []func(*BhWidget, ...interface{}){}
				}
				funcs = append(funcs, f)
			} else {
				if props == nil {
					props = []interface{}{}
				}
				props = append(props, propToTest)
			}
		}
	}
	return props, funcs
}

func (bhwdgt *BhWidget) SingleElem(elemname string, props ...interface{}) {
	bhwdgt.Out().Print("<", elemname)
	bhwdgt.printProps(props...)
	bhwdgt.Out().Print(">")
}

func (bhwdgt *BhWidget) EndElem(elemname string) {
	bhwdgt.Out().Print("</", elemname, ">")
}

//HTML
func (bhwdgt *BhWidget) Doctype() {
	bhwdgt.Out().Println("<!doctype html>")
}

func (bhwdgt *BhWidget) Html(htmlContent ...interface{}) {
	elmprops, elmfuncs := splitPropsIntoFuncAndProps(htmlContent...)
	bhwdgt.StartHtml(elmprops...)
	callElemFuncs(bhwdgt, elmfuncs)
	bhwdgt.EndHtml()
}

func (bhwdgt *BhWidget) HtmlComment(comments ...interface{}) {
	if len(comments) > 0 {
		bhwdgt.Out().Println("<!--")
		bhwdgt.Out().Println(comments)
		bhwdgt.Out().Println("-->")
	}
}

func (bhwdgt *BhWidget) StartHtml(props ...interface{}) {
	bhwdgt.StartElem("html")
}

func (bhwdgt *BhWidget) StartHead() {
	bhwdgt.StartElem("head")
}

func (bhwdgt *BhWidget) Head(headContents ...interface{}) {
	_, elmfuncs := splitPropsIntoFuncAndProps(headContents...)
	bhwdgt.StartHead()
	callElemFuncs(bhwdgt, elmfuncs)
	bhwdgt.EndHead()
}

func (bhwdgt *BhWidget) Script(src ...string) {
	bhwdgt.Out().Print("<script")
	if len(src) > 0 {
		for _, s := range src {
			bhwdgt.Out().Print(" src=\"" + s + "\"")
		}
	}
	bhwdgt.Out().Print(">")
}

func (bhwdgt *BhWidget) StartScript() {
	bhwdgt.StartElem("script")
}

func (bhwgdt *BhWidget) EndScript() {
	bhwgdt.EndElem("script")
}

func (bhwdgt *BhWidget) EndHead() {
	bhwdgt.EndElem("head")
}

func (bhwdgt *BhWidget) StartBody(props ...interface{}) {
	bhwdgt.StartElem("body", props...)
}

func (bhwdgt *BhWidget) Body(bodyContents ...interface{}) {
	elmprops, elmfuncs := splitPropsIntoFuncAndProps(bodyContents...)
	bhwdgt.StartBody(elmprops...)
	callElemFuncs(bhwdgt, elmfuncs)
	bhwdgt.EndBody()
}

func (bhwdgt *BhWidget) EndBody() {
	bhwdgt.EndElem("body")
}

func (bhwdgt *BhWidget) EndHtml() {
	bhwdgt.EndElem("html")
}

//BODY

func (bhwdgt *BhWidget) StartTable(props ...interface{}) {
	bhwdgt.StartElem("table")
}

func (bhwdgt *BhWidget) StartThead() {
	bhwdgt.StartElem("thead")
}

func (bhwdgt *BhWidget) EndThead() {
	bhwdgt.EndElem("thead")
}

func (bhwdgt *BhWidget) StartTh(props ...interface{}) {
	bhwdgt.StartElem("th", props...)
}

func (bhwdgt *BhWidget) EndTh() {
	bhwdgt.EndElem("th")
}

func (bhwdgt *BhWidget) StartTfoot() {
	bhwdgt.StartElem("tfoot")
}

func (bhwdgt *BhWidget) EndTfoot() {
	bhwdgt.EndElem("tfoot")
}

func (bhwdgt *BhWidget) StartTd(props ...interface{}) {
	bhwdgt.StartElem("td", props...)
}

func (bhwdgt *BhWidget) EndTd() {
	bhwdgt.EndElem("td")
}

func (bhwdgt *BhWidget) EndTable() {
	bhwdgt.EndElem("table")
}

func (bhwdgt *BhWidget) StartDiv(props ...interface{}) {
	bhwdgt.StartElem("div", props...)
}

func (bhwdgt *BhWidget) EndDiv() {
	bhwdgt.EndElem("div")
}

//DB QUERYING
func (bhwdgt *BhWidget) Query(name string, alias string, query string, call goja.Value, qryparams ...interface{}) {
	if call != nil {
		qryprops, qryfuncs := splitPropsIntoFuncAndProps(qryparams...)
		dbquery := bhwdgt.srvlt.Query(alias, query, qryprops...)
		if qryfuncs == nil {
			qryfuncs = []func(*BhWidget, ...interface{}){}
		}
		qryfuncs = append(qryfuncs, func(wdgt *BhWidget, a ...interface{}) {
			if name == "" {
				wdgt.srvlt.BhScript.ScriptedFunc(call, dbquery)
			} else {
				wdgt.srvlt.BhScript.ScriptedFunc(call, dbquery, name)
			}
		})

		//bhscript.CallScriptFunc(bhwdgt.srvlt.BhScript, call, dbquery)
		callElemFuncs(bhwdgt, qryfuncs)

	} else {
		qryprops, qryfuncs := splitPropsIntoFuncAndProps(qryparams...)
		dbquery := bhwdgt.srvlt.Query(alias, query, qryprops...)
		callElemFuncs(bhwdgt, qryfuncs, dbquery)
	}
}

func (bhwdgt *BhWidget) DataSet(id string, alias string, query string, datasetprops ...interface{}) {
	qryprops, _ := splitPropsIntoFuncAndProps(datasetprops...)
	dbquery := bhwdgt.srvlt.Query(alias, query, qryprops...)
	bhwdgt.StartDiv("id=" + id + "container")
	bhwdgt.EndDiv()
	bhwdgt.StartScript()
	dbquery.PrintResult(bhwdgt.Out(), id, ".js")
	bhwdgt.Out().Print(`$('#` + id + `container').html('<table id="` + id + `" class="display"><thead><th>'+dataset_` + id + `[0].join('</th><th>')+'</th></thead><tfoot><th>'+dataset_` + id + `[0].join('</th><th>')+'</th></tfoot></table>');
	$('#` + id + `container table').DataTable({
        data: dataset_` + id + `[1]
	});`)
	bhwdgt.EndScript()
}
