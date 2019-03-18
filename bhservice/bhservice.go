package bhservice

import (
	"strings"

	"github.com/kardianos/osext"
	"github.com/kardianos/service"
)

func (svr *BhService) Start(s service.Service) error {
	if svr.start != nil {
		if svr.isService {
			go svr.start(svr, svr.args...)
		} else if svr.isConsole {
			svr.start(svr, svr.args...)
		}
	}

	if svr.isService {
		go svr.exec()
	} else if svr.isConsole {
		svr.exec()
	}
	return nil
}

func (svr *BhService) exec() {
	if svr.run != nil {
		if svr.isService || svr.isConsole {
			svr.run(svr, svr.args...)
		}
	}
}

func (svr *BhService) Stop(s service.Service) error {
	if svr.stop != nil {
		if svr.isService || svr.isConsole {
			svr.stop(svr, svr.args...)
		}
	}
	return nil
}

type BhService struct {
	isService   bool
	isConsole   bool
	start       func(*BhService, ...string)
	run         func(*BhService, ...string)
	stop        func(*BhService, ...string)
	execname    string
	execfolder  string
	name        string
	displayName string
	description string
	svcConfig   *service.Config
	args        []string
}

func (svr *BhService) IsConsole() bool {
	return svr.isConsole
}

func (svr *BhService) IsService() bool {
	return svr.isService
}

func (svr *BhService) ServiceExeName() string {
	return svr.execname
}

func (svr *BhService) ServiceName() string {
	return svr.name
}

func (svr *BhService) ServiceExeFolder() string {
	return svr.execfolder
}

func (svr *BhService) ServiceDisplayName() string {
	return svr.displayName
}

func (svr *BhService) ServiceDescription() string {
	return svr.description
}

func NewService(name string, displayName string, description string, start func(*BhService, ...string),
	run func(*BhService, ...string),
	stop func(*BhService, ...string)) (svr *BhService, err error) {
	if run != nil {
		execname, _ := osext.Executable()
		execname = strings.Replace(execname, "\\", "/", -1)
		execfolder, _ := osext.ExecutableFolder()
		execfolder = strings.Replace(execfolder, "\\", "/", -1)
		if name == "" {
			if execname != "" && execfolder != "" {
				execname = execname[len(execfolder)+1:]
			}
			name = execname
			if si := strings.Index(name, "."); si > -1 {
				name = name[0:si]
			}
		}

		if displayName == "" {
			displayName = name
		}

		if description == "" {
			description = strings.ToUpper(displayName)
		}
		//svcargs := []string{}

		svcConfig := &service.Config{
			Name:        name,
			DisplayName: displayName,
			Description: description,
		}

		svr = &BhService{execfolder: execfolder, execname: execname, start: start, run: run, stop: stop, name: name, displayName: displayName, description: description, svcConfig: svcConfig}
	}
	return svr, err
}

var logger service.Logger

func (svr *BhService) Execute(args []string) (err error) {
	svcargs := []string{}
	canappendargs := false
	if len(args) > 0 {
		for _, arg := range args[1:] {
			if arg == "install" {
				canappendargs = true
			} else if strings.Index(",start,stop,restart,install,uninstall,console,", ","+arg+",") > -1 {
				canappendargs = false
				break
			} else {
				svcargs = append(svcargs, arg)
			}
		}
	}
	if canappendargs && len(svcargs) > 0 {
		svr.svcConfig.Arguments = svcargs
	}

	if s, serr := service.New(svr, svr.svcConfig); serr == nil {
		if logger, err = s.Logger(nil); err == nil {
			argFound := ""
			svr.args = args[:]
			for _, arg := range svr.args {
				if strings.Index(",start,stop,restart,install,uninstall,", ","+arg+",") > -1 {
					argFound = arg
					svr.isService = true
					if err = service.Control(s, argFound); err == nil {
						break
					}
				} else if strings.Index(",console,", ","+arg+",") > -1 {
					svr.isConsole = true
					break
				}
				if err != nil {
					break
				}
			}
			if err == nil && argFound == "" {
				if !svr.isService {
					svr.isService = !svr.isConsole
				}
				if svr.isService {
					err = s.Run()
				} else if svr.isConsole {
					svr.Start(s)
					svr.Stop(s)
				}
			}
		}
	} else {
		err = serr
	}

	if err != nil {
		logger.Error(err)
	}

	return err
}
