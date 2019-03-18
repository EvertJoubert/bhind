package bhservlet

import (
	"path/filepath"
	"sync"
)

type BhResourceModel struct {
	res        *BhResource
	isext      bool
	isjsext    bool
	pathedExts []string
}

func (resMod *BhResourceModel) prepairResourceHandler(resHndlr *BhResourceHandler) {

}

func newResourceModel(res *BhResource) (resModel *BhResourceModel) {
	resModel = &BhResourceModel{res: res, isext: res.isext, isjsext: res.isjsext, pathedExts: recModExtPaths().PathedExtensions(filepath.Ext(res.path))}
	return resModel
}

type ResourceModelingExtensionPathedExtensions map[string][]string

func (resModExtPathedExts ResourceModelingExtensionPathedExtensions) Extensions() (extensions []string) {
	for ext, _ := range resModExtPathedExts {
		extensions = append(extensions, ext)
	}
	return extensions
}

func (resModExtPathedExts ResourceModelingExtensionPathedExtensions) PathedExtensions(ext string) (pathedexts []string) {
	pathedexts, _ = resModExtPathedExts[ext]
	return pathedexts
}

func (resModExtPathedExts ResourceModelingExtensionPathedExtensions) AppendPathedExtension(extionsion string, pathedextions ...string) {
	if len(pathedextions) > 0 {
		extionsPaths := resModExtPathedExts.PathedExtensions(extionsion)
		if extionsPaths == nil {
			extionsPaths = []string{}
		}
		for _, pathext := range pathedextions {
			extionsPaths = append(extionsPaths, pathext)
		}
		resModExtPathedExts[extionsion] = pathedextions
	}
}

var resourceModelingExtensionPathedExts ResourceModelingExtensionPathedExtensions
var resModExtPathedExtsLock *sync.Mutex = &sync.Mutex{}

func resourceModExtPathedExts(ext string, defaultext string) (pathedExts []string) {
	if pathedExts == nil {
		pathedExts = []string{}
	}

	return pathedExts
}

func recModExtPaths() ResourceModelingExtensionPathedExtensions {
	if resourceModelingExtensionPathedExts == nil {
		resModExtPathedExtsLock.Lock()
		if resourceModelingExtensionPathedExts == nil {
			resourceModelingExtensionPathedExts = make(ResourceModelingExtensionPathedExtensions)
		}
		resModExtPathedExtsLock.Unlock()
	}
	return resourceModelingExtensionPathedExts
}
