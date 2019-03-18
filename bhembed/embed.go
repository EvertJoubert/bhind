package bhembed

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
)

func DataTablesCSS() io.Reader {
	return strings.NewReader(datatablescss)
}

func DataTablesJS(incbootstrap bool) io.Reader {
	if incbootstrap {
		return strings.NewReader(datatablesjs + datatablesbootstrapjs)
	} else {
		return strings.NewReader(datatablesjs)
	}
}

func FontAwesomeJS() io.Reader {
	return strings.NewReader(fontawesomejs)
}

func JsonDateExtensions() io.Reader {
	return strings.NewReader(jsondateexstensions)
}

func JQueryJS() io.Reader {
	return strings.NewReader(jqueryminjs)
}

func JQueryUiJS() io.Reader {
	return strings.NewReader(strings.Replace(jqueryuijs, "|'|", "`", -1))
}

func JQueryUiCSS() io.Reader {
	return strings.NewReader(jqueryuicss)
}

func JQueryUiImages(image string) io.Reader {
	if strings.HasSuffix(image, "ui-icons_444444_256x240.png") {
		if decoded, err := base64.StdEncoding.DecodeString(ui_icons_444444_256x240png); err == nil {
			return bytes.NewReader(decoded)
		}
	} else if strings.HasSuffix(image, "ui-icons_555555_256x240.png") {
		if decoded, err := base64.StdEncoding.DecodeString(ui_icons_555555_256x240png); err == nil {
			return bytes.NewReader(decoded)
		}
	} else if strings.HasSuffix(image, "ui-icons_777620_256x240.png") {
		if decoded, err := base64.StdEncoding.DecodeString(ui_icons_777620_256x240png); err == nil {
			return bytes.NewReader(decoded)
		}
	} else if strings.HasSuffix(image, "ui-icons_777777_256x240.png") {
		if decoded, err := base64.StdEncoding.DecodeString(ui_icons_777777_256x240png); err == nil {
			return bytes.NewReader(decoded)
		}
	} else if strings.HasSuffix(image, "ui-icons_cc0000_256x240.png") {
		if decoded, err := base64.StdEncoding.DecodeString(ui_icons_cc0000_256x240png); err == nil {
			return bytes.NewReader(decoded)
		}
	} else if strings.HasSuffix(image, "ui-icons_ffffff_256x240.png") {
		if decoded, err := base64.StdEncoding.DecodeString(ui_icons_ffffff_256x240png); err == nil {
			return bytes.NewReader(decoded)
		}
	}
	return strings.NewReader("")
}

func JQueryUiTestHtml() io.Reader {
	return strings.NewReader(jquerytesthtml)
}

func BootstrapJS() io.Reader {
	return strings.NewReader(bootstrapjs)
}

func BootstrapCSS() io.Reader {
	return strings.NewReader(bootstrapcss)
}

func BootstrapDatepickerCSS() io.Reader {
	return strings.NewReader(bootstrapdatepickercss)
}

func BootstrapDatepickerJS() io.Reader {
	return strings.NewReader(strings.Replace(bootstrapdatepickerjs, "|'|", "`", -1))
}

func VueJS() io.Reader {
	return strings.NewReader(strings.Replace(vueminjs, "|'|", "`", -1))
}

func BabylonJS() io.Reader {
	return strings.NewReader(babylonjs)
}

func VueBabylonJS() io.Reader {
	return strings.NewReader(vuebabylonjs)
}

func VueBootstrapCSS() io.Reader {
	return strings.NewReader(vuebootstrapcss)
}

func VueBootstrapJS(inclvue bool) io.Reader {
	if inclvue {
		return strings.NewReader(strings.Replace(vueminjs, "|'|", "`", -1) + strings.Replace(vuebootstrapjs, "|'|", "`", -1))
	} else {
		return strings.NewReader(strings.Replace(vuebootstrapjs, "|'|", "`", -1))
	}
}

func WebActionsJS() io.Reader {
	return strings.NewReader(webactionsjs)
}

func BlockUiJS() io.Reader {
	return strings.NewReader(blockuijs)
}

func EmbeddedList(resources ...string) (a []interface{}) {
	a = []interface{}{}
	if len(resources) == 0 {
		a = append(a,
			"json.date-extensions.js", JsonDateExtensions(),
			"webactions.js", WebActionsJS(),
			"vue.js", VueJS(),
			"jquery.js", JQueryJS(),
			"blockui.js", BlockUiJS(),
			"babylon.js", BabylonJS(),
			"fontawesome.js", FontAwesomeJS(),
			"datatables.css", DataTablesCSS(),
			"datatables.js", DataTablesJS(false),
			"datatables-bootstrap.js", DataTablesJS(true),
			"bootstrap.js", BootstrapJS(),
			"bootstrap.css", BootstrapCSS(),
			"bootstrap-datepicker.js", BootstrapDatepickerJS(),
			"bootstrap-datepicker.css", BootstrapDatepickerCSS(),
			"vue-babylon.js", VueBabylonJS(),
			"vue-bootstrap.js", VueBootstrapJS(false),
			"vue-incl-bootstrap.js", VueBootstrapJS(true),
			"vue-bootstrap.css", VueBootstrapCSS(),
			"jquery-ui.js", JQueryUiJS(),
			"jquery-ui.css", JQueryUiCSS(),
			"jquery-ui-test.html", JQueryUiTestHtml(),
			"ui-icons_444444_256x240.png", JQueryUiImages("ui-icons_444444_256x240.png"),
			"ui-icons_555555_256x240.png", JQueryUiImages("ui-icons_555555_256x240.png"),
			"ui-icons_777777_256x240.png", JQueryUiImages("ui-icons_777777_256x240.png"),
			"ui-icons_777620_256x240.png", JQueryUiImages("ui-icons_777620_256x240.png"),
			"ui-icons_cc0000_256x240.png", JQueryUiImages("ui-icons_cc0000_256x240.png"),
			"ui-icons_ffffff_256x240.png", JQueryUiImages("ui-icons_ffffff_256x240.png"))
	} else {
		for _, resname := range resources {
			if resname == "json.date-extensions.js" {
				a = append(a, "json.date-extensions.js", JsonDateExtensions())
			} else if resname == "webactions.js" {
				a = append(a, "webactions.js", WebActionsJS())
			} else if resname == "vue.js" {
				a = append(a, "vue.js", VueJS())
			} else if resname == "jquery.js" {
				a = append(a, "jquery.js", JQueryJS())
			} else if resname == "blockui.js" {
				a = append(a, "blockui.js", BlockUiJS())
			} else if resname == "babylon.js" {
				a = append(a, "babylon.js", BabylonJS())
			} else if resname == "fontawesome.js" {
				a = append(a, "fontawesome.js", FontAwesomeJS())
			} else if resname == "bootstrap.js" {
				a = append(a, "bootstrap.js", BootstrapJS())
			} else if resname == "bootstrap.css" {
				a = append(a, "bootstrap.css", BootstrapCSS())
			} else if resname == "datatables.css" {
				a = append(a, "datatables.css", DataTablesCSS())
			} else if resname == "datatables.js" {
				a = append(a, "datatables.js", DataTablesJS(false))
			} else if resname == "datatables-bootstrap.js" {
				a = append(a, "datatables-bootstrap.js", DataTablesJS(true))
			} else if resname == "bootstrap-datepicker.js" {
				a = append(a, "bootstrap-datepicker.js", BootstrapDatepickerJS())
			} else if resname == "bootstrap-datepicker.css" {
				a = append(a, "bootstrap-datepicker.css", BootstrapDatepickerCSS())
			} else if resname == "vue-babylon.js" {
				a = append(a, "vue-babylon.js", VueBabylonJS())
			} else if resname == "vue-bootstrap.js" {
				a = append(a, "vue-bootstrap.js", VueBootstrapJS(false))
			} else if resname == "vue-incl-bootstrap.js" {
				a = append(a, "vue-incl-bootstrap.js", VueBootstrapJS(true))
			} else if resname == "vue-bootstrap.css" {
				a = append(a, "vue-bootstrap.css", VueBootstrapCSS())
			} else if resname == "jquery-ui.js" {
				a = append(a, "jquery-ui.js", JQueryUiJS())
			} else if resname == "jquery-ui.css" {
				a = append(a, "jquery-ui.css", JQueryUiCSS())
			} else if resname == "jquery-ui-test.html" {
				a = append(a, "jquery-ui-test.html", JQueryUiTestHtml())
			} else if resname == "jquery-ui-images.png" {
				a = append(a,
					"ui-icons_444444_256x240.png", JQueryUiImages("ui-icons_444444_256x240.png"),
					"ui-icons_555555_256x240.png", JQueryUiImages("ui-icons_555555_256x240.png"),
					"ui-icons_777777_256x240.png", JQueryUiImages("ui-icons_777777_256x240.png"),
					"ui-icons_777620_256x240.png", JQueryUiImages("ui-icons_777620_256x240.png"),
					"ui-icons_cc0000_256x240.png", JQueryUiImages("ui-icons_cc0000_256x240.png"),
					"ui-icons_ffffff_256x240.png", JQueryUiImages("ui-icons_ffffff_256x240.png"))
			}
		}
	}
	return a
}
