package webx

var plugins []*Plugin

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

func (plugin *Plugin) Page(name string, buf []byte) {
	plugin.pages[name] = buf
}

func (plugin *Plugin) Asset(name string, buf []byte) {
	plugin.assets[name] = buf
}
