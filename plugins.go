package webx

import "path/filepath"

var plugins = []*Plugin{}
var pluginPaths = map[string][]string{}

type Plugin struct {
	name   string
	config map[string]string
	router []func(app App)
	pages  map[string][]byte
	assets map[string][]byte
}

func NewPlugin(name string, defConfig ...map[string]string) *Plugin {
	if len(defConfig) == 0 {
		defConfig = append(defConfig, map[string]string{})
	}

	plugin := &Plugin{
		name:   name,
		config: defConfig[0],
		router: []func(app App){},
		pages:  map[string][]byte{},
		assets: map[string][]byte{},
	}

	plugins = append(plugins, plugin)

	return plugin
}

func (plugin *Plugin) Router(cb func(app App)) {
	plugin.router = append(plugin.router, cb)
}

func (plugin *Plugin) Page(name string, buf []byte, path ...string) {
	plugin.pages[name] = buf

	if len(path) != 0 {
		if p, err := filepath.Abs(path[0]); err == nil {
			pluginPaths[p] = []string{name, plugin.name}
		}
	}
}

func (plugin *Plugin) Asset(name string, buf []byte, path ...string) {
	plugin.assets[name] = buf

	if len(path) != 0 {
		if p, err := filepath.Abs(path[0]); err == nil {
			pluginPaths[p] = []string{name, plugin.name}
		}
	}
}
