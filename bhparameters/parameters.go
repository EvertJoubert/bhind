package bhparameters

import (
	"io"
	"mime/multipart"
	"strings"
)

//Parameters -> structure containing parameters
type Parameters struct {
	standard  map[string][]string
	filesdata map[string][]interface{}
}

var emptyParmVal = []string{}
var emptyParamFile = []interface{}{}

/*	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer*/

//SetParameter -> set or append parameter value
//pname : name
//pvalue : value of strings to add
//clear : clear existing value of parameter
func (params *Parameters) SetParameter(pname string, clear bool, pvalue ...string) {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return
	}
	if params.standard == nil {
		params.standard = make(map[string][]string)
	}
	if val, ok := params.standard[pname]; ok {
		if clear {
			val = nil
			params.standard[pname] = nil
			val = []string{}
		}
		if len(pvalue) > 0 {
			val = append(val, pvalue...)
		}
		params.standard[pname] = val
	} else {
		if len(pvalue) > 0 {
			params.standard[pname] = pvalue[:]
		} else {
			params.standard[pname] = []string{}
		}
	}
}

//ContainsParameter -> check if parameter exist
//pname : name
func (params *Parameters) ContainsParameter(pname string) bool {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return false
	}
	if params.standard == nil {
		return false
	}
	_, ok := params.standard[pname]
	return ok
}

//SetFileParameter -> set or append file parameter value
//pname : name
//pfile : value of interface to add either FileHeader from mime/multipart or any io.Reader implementation
//clear : clear existing value of parameter
func (params *Parameters) SetFileParameter(pname string, clear bool, pfile ...interface{}) {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return
	}
	if params.filesdata == nil {
		params.filesdata = make(map[string][]interface{})
	}
	if val, ok := params.filesdata[pname]; ok {
		if clear {
			val = nil
			params.filesdata[pname] = nil
			val = []interface{}{}
		}
		if len(pfile) > 0 {
			for _, pf := range pfile {
				if fheader, fheaderok := pf.(multipart.FileHeader); fheaderok {
					if fv, fverr := fheader.Open(); fverr == nil {
						if rval, rvalok := fv.(io.Reader); rvalok {
							val = append(val, rval)
						}
					}
				} else {
					val = append(val, pf)
				}
			}
		}
		params.filesdata[pname] = val
	} else {
		if len(pfile) > 0 {
			val = []interface{}{}
			for _, pf := range pfile {
				if fheader, fheaderok := pf.(multipart.FileHeader); fheaderok {
					if fv, fverr := fheader.Open(); fverr == nil {
						if rval, rvalok := fv.(io.Reader); rvalok {
							val = append(val, rval)
						}
					}
				} else {
					val = append(val, pf)
				}
			}
			params.filesdata[pname] = val
		} else {
			params.filesdata[pname] = []interface{}{}
		}
	}
}

//ContainsFileParameter -> check if file parameter exist
//pname : name
func (params *Parameters) ContainsFileParameter(pname string) bool {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname == "" {
		return false
	}
	if params.filesdata == nil {
		return false
	}
	_, ok := params.filesdata[pname]
	return ok
}

//Parameter - return a specific parameter values
func (params *Parameters) Parameter(pname string) []string {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname != "" {
		if params.standard != nil {
			if _, ok := params.standard[pname]; ok {
				return params.standard[pname]
			}
		}
	}
	return emptyParmVal
}

func (params *Parameters) StringParameter(pname string, sep string) string {
	if pval := params.Parameter(pname); len(pval) > 0 {
		return strings.Join(pval, sep)
	} else {
		return ""
	}
}

func (params *Parameters) FileParameter(pname string) []interface{} {
	if pname = strings.ToUpper(strings.TrimSpace(pname)); pname != "" {
		if params.filesdata != nil {
			if _, ok := params.filesdata[pname]; ok {
				return params.filesdata[pname]
			}
		}
	}
	return emptyParamFile
}

func (params *Parameters) CleanupParameters() {
	if params.standard != nil {
		for pname := range params.standard {
			params.standard[pname] = nil
			delete(params.standard, pname)
		}
		params.standard = nil
	}
	if params.filesdata != nil {
		for pname := range params.filesdata {
			params.filesdata[pname] = nil
			delete(params.filesdata, pname)
		}
		params.filesdata = nil
	}
}

func NewParameters() *Parameters {
	return &Parameters{}
}
